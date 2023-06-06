package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type RedisPubSubAdapter struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisPubSubAdapter(addr string) (*RedisPubSubAdapter, error) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisPubSubAdapter{
		client: client,
		ctx:    ctx,
	}, nil
}

func (r *RedisPubSubAdapter) Publish(topicARN string, message interface{}, source string, messageAttributes map[string]interface{}) error {
	if source == "" {
		return fmt.Errorf("source cannot be empty")
	}
	messageAttributes["source"] = source
	messageWithAtrributs := map[string]interface{}{
		"messageAttributs": messageAttributes,
		"message":          message,
	}
	err := r.client.Publish(r.ctx, topicARN, messageWithAtrributs).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisPubSubAdapter) PollMessages(topic string, handler func(message string)) error {
	pubsub := r.client.Subscribe(r.ctx, topic)
	defer pubsub.Close()

	_, err := pubsub.ReceiveMessage(r.ctx)
	if err != nil {
		return err
	}

	channel := pubsub.Channel()
	for msg := range channel {
		handler(string(*&msg.Payload))
	}
	return nil
}
