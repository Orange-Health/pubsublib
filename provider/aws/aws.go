package aws

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	pubsub "github.com/Orange-Health/pubsublib"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/aws/aws-sdk-go/service/sqs/sqsiface"
)

type AWSPubSubAdapter struct {
	session *session.Session
	snsSvc  *sns.SNS
	sqsSvc  sqsiface.SQSAPI
}

func NewAWSPubSubAdapter(region, accessKeyId, secretAccessKey string) (*AWSPubSubAdapter, error) {
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

func (ps *AWSPubSubAdapter) Publish(topicARN string, message interface{}, attributeName string, attributeValue string) error {
	b, _ := message.(string)
	_, err := ps.snsSvc.Publish(&sns.PublishInput{
		Message:  aws.String(b),
		TopicArn: aws.String(topicARN),
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			attributeName: {
				DataType:    aws.String("String"),
				StringValue: aws.String(attributeValue),
			},
		},
	})
	return err
}

// not using this for v1
func (ps *AWSPubSubAdapter) Subscribe(topicARN string, handler pubsub.MessageHandler) error {
	subscribeOutput, err := ps.snsSvc.Subscribe(&sns.SubscribeInput{
		Protocol: aws.String("sqs"),
		Endpoint: aws.String(topicARN),
		TopicArn: aws.String(topicARN),
	})

	if err != nil {
		return err
	}
	subscriptionARN := *subscribeOutput.SubscriptionArn

	go ps.PollMessages(topicARN, handler)

	// Wait for termination signals to unsubscribe and cleanup
	ps.waitForTermination(topicARN, &subscriptionARN)

	return nil
}

func (ps *AWSPubSubAdapter) PollMessages(topicARN string, handler pubsub.MessageHandler) {
	for {
		result, err := ps.sqsSvc.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(topicARN),
			MaxNumberOfMessages: aws.Int64(10),
			VisibilityTimeout:   aws.Int64(5),
			WaitTimeSeconds:     aws.Int64(20),
		})

		if err != nil {
			log.Println("Error receiving message:", err)
			continue
		}

		for _, message := range result.Messages {
			err := handler([]byte(*message.Body))
			if err != nil {
				log.Println("Error handling message:", err)
				continue
			}

			_, err = ps.sqsSvc.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      aws.String(topicARN),
				ReceiptHandle: message.ReceiptHandle,
			})

			if err != nil {
				log.Println("Error deleting message:", err)
			}
		}
	}
}

func (ps *AWSPubSubAdapter) waitForTermination(topicARN string, subscriptionARN *string) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh // Wait for termination signal

	// Unsubscribe from the topic
	_, err := ps.snsSvc.Unsubscribe(&sns.UnsubscribeInput{
		SubscriptionArn: subscriptionARN,
	})
	if err != nil {
		log.Println("Error unsubscribing from the topic:", err)
	}

	// Delete the SQS queue
	_, err = ps.sqsSvc.DeleteQueue(&sqs.DeleteQueueInput{
		QueueUrl: aws.String(topicARN),
	})
	if err != nil {
		log.Println("Error deleting the queue:", err)
	}

	os.Exit(0) // Terminate the program
}
