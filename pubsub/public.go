package pubsub

// Publish published on the client
func (c *Client) Publish(topicARN string, message interface{}, attributeName string, attributeValue string) error {
	mw := chainPublisherMiddleware(c.Middleware...)
	return mw(c.ServiceName, func(topicARN string, message interface{}, attributeName string, attributeValue string) error {
		return c.Provider.Publish(topicARN, message, attributeName, attributeValue)
	})(topicARN, message, attributeName, attributeValue)
}

func chainPublisherMiddleware(mw ...Middleware) func(serviceName string, next PublishHandler) PublishHandler {
	return func(serviceName string, final PublishHandler) PublishHandler {
		return func(topicARN string, message interface{}, attributeName string, attributeValue string) error {
			last := final
			for i := len(mw) - 1; i >= 0; i-- {
				last = mw[i].PublisherMsgInterceptor(serviceName, last)
			}
			return last(topicARN, message, attributeName, attributeValue)
		}
	}
}
