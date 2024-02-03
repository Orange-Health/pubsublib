package aws

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Orange-Health/pubsublib/helper"
	"github.com/Orange-Health/pubsublib/infrastructure"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/google/uuid"
)

type AWSPubSubAdapter struct {
	session     *session.Session
	snsSvc      *sns.SNS
	sqsSvc      sqsiface.SQSAPI
	redisClient *infrastructure.RedisDatabase
}

func NewAWSPubSubAdapter(region, accessKeyId, secretAccessKey, redisAddress, redisPassword, snsEndpoint string, redisDB int) (*AWSPubSubAdapter, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:   aws.String(region),
		Endpoint: aws.String(snsEndpoint),
		Credentials: credentials.NewStaticCredentials(
			accessKeyId,
			secretAccessKey,
			"", // a token will be created when the session it's used.
		),
	})
	if err != nil {
		return nil, err
	}

	snsSvc := sns.New(sess)
	sqsSvc := sqs.New(sess)
	redisClient, err := infrastructure.NewRedisDatabase(redisAddress, redisPassword, redisDB)
	if err != nil {
		return nil, err
	}

	return &AWSPubSubAdapter{
		session:     sess,
		snsSvc:      snsSvc,
		sqsSvc:      sqsSvc,
		redisClient: redisClient,
	}, nil
}

/*
Publishes the message with the messageAttributes to the topicARN provided.
source, contains and eventType are necessary keys in messageAttributes.
Returns error if fails to publish message
*/
func (ps *AWSPubSubAdapter) Publish(topicARN string, message interface{}, messageAttributes map[string]interface{}) error {
	// Check if message is of type map[string]interface{} and then convert all the keys to snake_case
	switch message.(type) {
	case map[string]interface{}:
		message = helper.ConvertBodyToSnakeCase(message.(map[string]interface{}))
	}

	jsonString, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// figure out the message body as required
	messageBody := string(jsonString)
	if len(messageBody) > 200*1024 {
		// body is larger than 200kB. Best to put it in redis with expiry time of 10 days
		redisKey := uuid.New().String()
		messageAttributes["redis_key"] = redisKey

		// Set the message body in redis db
		err := ps.redisClient.Set(redisKey, messageBody, 10*24*60)
		if err != nil {
			return err
		}

		messageBody = "body is stored in redis under key PUBSUB:" + redisKey
	}

	if messageAttributes["source"] == nil {
		return fmt.Errorf("should have source key in messageAttributes")
	}
	if messageAttributes["contains"] == nil {
		return fmt.Errorf("should have contains key in messageAttributes")
	}
	if messageAttributes["event_type"] == nil {
		return fmt.Errorf("should have event_type key in messageAttributes")
	}
	if messageAttributes["trace_id"] == nil {
		messageAttributes["trace_id"] = uuid.New().String()
	}
	awsMessageAttributes := map[string]*sns.MessageAttributeValue{}
	if messageAttributes != nil {
		awsMessageAttributes, _ = BindAttributes(messageAttributes)
	}
	_, err = ps.snsSvc.Publish(&sns.PublishInput{
		Message:           aws.String(messageBody), // Ensures to always send compressed message
		TopicArn:          aws.String(topicARN),
		MessageAttributes: awsMessageAttributes,
	})
	if err != nil {
		return err
	}
	return nil
}

/*
Publishes the message with the messageAttributes to the FIFO enabled topicARN provided.
source, contains and eventType are necessary keys in messageAttributes.
Returns error if fails to publish message
*/
func (ps *AWSPubSubAdapter) PublishFIFO(topicARN, messageGroupId string, message interface{}, messageAttributes map[string]interface{}) error {
	// Check if message is of type map[string]interface{} and then convert all the keys to snake_case
	switch message.(type) {
	case map[string]interface{}:
		message = helper.ConvertBodyToSnakeCase(message.(map[string]interface{}))
	}

	jsonString, err := json.Marshal(message)
	if err != nil {
		return err
	}

	// figure out the message body as required
	messageBody := string(jsonString)
	if len(messageBody) > 200*1024 {
		// body is larger than 200kB. Best to put it in redis with expiry time of 10 days
		redisKey := uuid.New().String()
		messageAttributes["redis_key"] = redisKey

		// Set the message body in redis db
		err := ps.redisClient.Set(redisKey, messageBody, 10*24*60)
		if err != nil {
			return err
		}

		messageBody = "body is stored in redis under key PUBSUB:" + redisKey
	}

	if messageAttributes["source"] == nil {
		return fmt.Errorf("should have source key in messageAttributes")
	}
	if messageAttributes["contains"] == nil {
		return fmt.Errorf("should have contains key in messageAttributes")
	}
	if messageAttributes["event_type"] == nil {
		return fmt.Errorf("should have event_type key in messageAttributes")
	}
	if messageAttributes["trace_id"] == nil {
		messageAttributes["trace_id"] = uuid.New().String()
	}
	awsMessageAttributes := map[string]*sns.MessageAttributeValue{}
	if messageAttributes != nil {
		awsMessageAttributes, _ = BindAttributes(messageAttributes)
	}
	_, err = ps.snsSvc.Publish(&sns.PublishInput{
		Message:           aws.String(messageBody), // Ensures to always send compressed message
		TopicArn:          aws.String(topicARN),
		MessageAttributes: awsMessageAttributes,
		messageGroupId:    aws.String(messageGroupId),
	})
	if err != nil {
		return err
	}
	return nil
}

/*
Polls messages from SQS with queueURL, using long polling for 20 seconds, visibility timeout of 5 seconds and maximum of 10 messages read at once.
Handler func will be executed for each message individually, if error returned from the handler func is nil, message is deleted from queue, else returns error
*/
func (ps *AWSPubSubAdapter) PollMessages(queueURL string, handler func(message *sqs.Message) error) error {
	result, err := ps.sqsSvc.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: aws.Int64(10),
		VisibilityTimeout:   aws.Int64(5),
		WaitTimeSeconds:     aws.Int64(20),
		MessageAttributeNames: []*string{
			aws.String("All"),
		},
		AttributeNames: []*string{
			aws.String("All"),
		},
	})

	if err != nil {
		return err
	}

	for _, message := range result.Messages {
		// Verify the message integrity
		if !verifyMessageIntegrity(*message.Body, *message.MD5OfBody, message.MessageAttributes, *message.MD5OfMessageAttributes) {
			return fmt.Errorf("message corrupted")
		}
		if redisKey, ok := message.MessageAttributes["redis_key"]; ok {
			if messageBody, err := ps.FetchValueFromRedis(*redisKey.StringValue); err != nil {
				return err
			} else {
				message.Body = aws.String(messageBody)
			}
		}

		err = handler(message)
		if err != nil {
			return err
		}

		_, err = ps.sqsSvc.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      aws.String(queueURL),
			ReceiptHandle: message.ReceiptHandle,
		})

		if err != nil {
			return err
		}
	}
	return nil
}

// Expects the redis key (uuid), fetches the value from redis and returns it as string always. Since all values are stored as string in redis for now.
func (ps *AWSPubSubAdapter) FetchValueFromRedis(redisKey string) (string, error) {
	var messageBody string
	err := ps.redisClient.Get(redisKey, &messageBody)
	if err != nil {
		return "", err
	}
	return messageBody, nil
}

// not using this for v1
// func (ps *AWSPubSubAdapter) Subscribe(topicARN string, handler func(message string) error) error {
// 	subscribeOutput, err := ps.snsSvc.Subscribe(&sns.SubscribeInput{
// 		Protocol: aws.String("sqs"),
// 		Endpoint: aws.String(topicARN),
// 		TopicArn: aws.String(topicARN),
// 	})

// 	if err != nil {
// 		return err
// 	}
// 	subscriptionARN := *subscribeOutput.SubscriptionArn

// 	go ps.PollMessages(topicARN, handler)

// 	// Wait for termination signals to unsubscribe and cleanup
// 	ps.waitForTermination(topicARN, &subscriptionARN)

// 	return nil
// }

// func (ps *AWSPubSubAdapter) waitForTermination(topicARN string, subscriptionARN *string) {
// 	sigCh := make(chan os.Signal, 1)
// 	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

// 	<-sigCh // Wait for termination signal

// 	// Unsubscribe from the topic
// 	_, err := ps.snsSvc.Unsubscribe(&sns.UnsubscribeInput{
// 		SubscriptionArn: subscriptionARN,
// 	})
// 	if err != nil {
// 		log.Println("Error unsubscribing from the topic:", err)
// 	}

// 	// Delete the SQS queue
// 	_, err = ps.sqsSvc.DeleteQueue(&sqs.DeleteQueueInput{
// 		QueueUrl: aws.String(topicARN),
// 	})
// 	if err != nil {
// 		log.Println("Error deleting the queue:", err)
// 	}

// 	os.Exit(0) // Terminate the program
// }

func BindAttributes(attributes map[string]interface{}) (map[string]*sns.MessageAttributeValue, error) {
	boundAttributes := make(map[string]*sns.MessageAttributeValue)

	for key, value := range attributes {
		attrValue, _ := convertToAttributeValue(value)
		boundAttributes[key] = attrValue
	}
	return boundAttributes, nil
}

func convertToAttributeValue(value interface{}) (*sns.MessageAttributeValue, error) {
	// Perform type assertions or conversions based on the expected types of attributes
	// and create the appropriate sns.MessageAttributeValue object.

	switch v := value.(type) {
	case string:
		return &sns.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(v),
		}, nil
	case int, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return &sns.MessageAttributeValue{
			DataType:    aws.String("Number"),
			StringValue: aws.String(fmt.Sprint(v)),
		}, nil
	case []string:
		return &sns.MessageAttributeValue{
			DataType:    aws.String("String.Array"),
			StringValue: aws.String(strings.Join(v, ",")),
		}, nil
	// Add more cases for other data types as needed

	default:
		return nil, fmt.Errorf("unsupported attribute value type: %T", value)
	}
}

/*
Compares the calculated MD5 hashes with the received MD5 hashes.
If the MD5 hashes match, the message is not corrupted hence returns true
*/
func verifyMessageIntegrity(messageBody, md5OfBody string, messageAttributes map[string]*sqs.MessageAttributeValue, md5OfMessageAttributes string) bool {
	// Calculate the MD5 hash of the message body
	calculatedMD5OfBody := calculateMD5Hash(messageBody)

	// Compare the calculated MD5 hashes with the received MD5 hashes
	return calculatedMD5OfBody == md5OfBody
}

// Calculates the MD5 hash of the data passed
func calculateMD5Hash(data string) string {
	hasher := md5.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}
