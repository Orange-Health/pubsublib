package pubsub

// NoopProvider is a simple provider that does nothing, for testing, defaults
type NoopProvider struct{}

// Publish does nothing
func (np NoopProvider) Publish(topicARN string, message interface{}, source string, messageAttributes map[string]interface{}) error {
	return nil
}

// Subscribe does nothing
func (np NoopProvider) PollMessages(queueURL string, handler MessageHandler) error {
	return nil
}
