package aws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Orange-Health/pubsublib/helper"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
	"github.com/google/uuid"
)

type AWSPubSubAdapter struct {
	session *session.Session
	snsSvc  *sns.SNS
	sqsSvc  sqsiface.SQSAPI
}

func NewAWSPubSubAdapter(region string, accessKeyId string, secretAccessKey string) (*AWSPubSubAdapter, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
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

	return &AWSPubSubAdapter{
		session: sess,
		snsSvc:  snsSvc,
		sqsSvc:  sqsSvc,
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
	if messageAttributes["source"] == nil {
		return fmt.Errorf("should have source key in messageAttributes")
	}
	if messageAttributes["contains"] == nil {
		return fmt.Errorf("should have contains key in messageAttributes")
	}
	if messageAttributes["event_type"] == nil {
		return fmt.Errorf("should have eventType key in messageAttributes")
	}
	if messageAttributes["trace_id"] == nil {
		messageAttributes["trace_id"] = uuid.New().String()
	}
	awsMessageAttributes := map[string]*sns.MessageAttributeValue{}
	if messageAttributes != nil {
		awsMessageAttributes, _ = BindAttributes(messageAttributes)
	}
	_, err = ps.snsSvc.Publish(&sns.PublishInput{
		Message:           aws.String(string(jsonString)),
		TopicArn:          aws.String(topicARN),
		MessageAttributes: awsMessageAttributes,
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
		err := handler(message)
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
