package awspubsub

import (
	"github.com/Orange-Health/pubsublib/pubsub"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type AWSPubSubAdapter struct {
	snsClient *sns.SNS
	sqsClient *sqs.SQS
}

func NewAWSPubSubAdapter(region, accessKeyId, secretAccessKey string) (pubsub.PubSub, error) {
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

	snsClient := sns.New(sess)
	sqsClient := sqs.New(sess)

	return &AWSPubSubAdapter{
		snsClient: snsClient,
		sqsClient: sqsClient,
	}, nil
}

func (a *AWSPubSubAdapter) Publish(topicARN string, message []byte) error {
	input := &sns.PublishInput{
		Message:  aws.String(string(message)),
		TopicArn: aws.String(topicARN),
	}

	_, err := a.snsClient.Publish(input)
	return err
}

func (a *AWSPubSubAdapter) Subscribe(queueURL string, handler func(msg []byte)) error {
	for {
		input := &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			WaitTimeSeconds:     aws.Int64(20),
			MaxNumberOfMessages: aws.Int64(10),
		}

		result, err := a.sqsClient.ReceiveMessage(input)
		if err != nil {
			return err
		}

		// TODO: need to decide if this is how we should handle messages that
		// have been received.
		for _, message := range result.Messages {
			handler([]byte(*message.Body))

			deleteInput := &sqs.DeleteMessageInput{
				QueueUrl:      aws.String(queueURL),
				ReceiptHandle: message.ReceiptHandle,
			}

			_, err := a.sqsClient.DeleteMessage(deleteInput)
			if err != nil {
				return err
			}
		}
	}
}
