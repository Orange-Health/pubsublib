package pubsub

type MessageHandler func(message []byte) error

var (
	clients = []*Client{&Client{Provider: NoopProvider{}}}
)

// Client holds a reference to a Provider
type Client struct {
	ServiceName string
	Provider    Provider
	Middleware  []Middleware
}

type Provider interface {
	Publish(topicARN string, message interface{}, attributeName string, attributeValue string) error
	PollMessages(queueURL string, handler MessageHandler) error
}

// SetClient sets the global pubsub client, useful in tests
func SetClient(cli *Client) {
	clients = []*Client{cli}
}

// PublishHandler wraps a call to publish, for interception
type PublishHandler func(topicARN string, message interface{}, attributeName string, attributeValue string) error

// Middleware is an interface to provide subscriber and publisher interceptors
type Middleware interface {
	PublisherMsgInterceptor(serviceName string, next PublishHandler) PublishHandler
}
