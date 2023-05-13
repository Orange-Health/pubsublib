package redispubsub

import (
	"context"

	"github.com/Orange-Health/pubsublib/pubsub"
	"github.com/redis/go-redis/v9"
)

type RedisPubSubAdapter struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisPubSubAdapter(addr string) (pubsub.PubSub, error) {
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisPubSubAdapter{
		client: client,
		ctx:    ctx,
	}, nil
}

func (r *RedisPubSubAdapter) Publish(topic string, message []byte) error {
	return r.client.Publish(r.ctx, topic, message).Err()
}

func (r *RedisPubSubAdapter) Subscribe(topic string, handler func(msg []byte)) error {
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
