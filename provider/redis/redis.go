package redis

import (
	"context"

	pubsub "github.com/Orange-Health/pubsublib"
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

func (r *RedisPubSubAdapter) Publish(topicARN string, message interface{}, attributeName string, attributeValue string) error {
	err := r.client.Publish(r.ctx, topicARN, message).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisPubSubAdapter) PollMessages(topic string, handler pubsub.MessageHandler) error {
	pubsub := r.client.Subscribe(r.ctx, topic)
	defer pubsub.Close()

	_, err := pubsub.ReceiveMessage(r.ctx)
	if err != nil {
		return err
	}

	channel := pubsub.Channel()
	for msg := range channel {
		handler([]byte(msg.Payload))
	}
	return nil
}
