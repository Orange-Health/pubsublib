package pubsub

type PubSub interface {
	Publish(topic string, message []byte) error
	Subscribe(topic string, handler func(msg []byte)) error
}
