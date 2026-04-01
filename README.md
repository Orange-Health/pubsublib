# pubsublib
Common pubsub library to be used across OH libraries

## Getting Started

To use pubsublib in your golang service use

```sh
go get -u github.com/Orange-Health/pubsublib@v0.3.3
```

### Quick Start

Import package
```go
	pubsub "github.com/Orange-Health/pubsublib/provider/aws"
```

and initialize 

```go

    
	awsAdapter, err := pubsub.NewAWSPubSubAdapter(
		"aws-region",
		"SQSAccessKeyID",
		"SQSSecretAccessKey",
		"SQSEndpointOverride",
		"PubSubRedisHost",
		"PubSubRedisPass",
		PubSubRedisDb,
		PubSubRedisPoolSize,
		PubSubRedisMinIdleConn,
	)
	if err != nil {
		errorService.LogError("Error while creating AWS adapter", err, nil)
		return
	}

	awsAdapter.SetCompressionEnabled(true)
```

#### Using AWS IAM Roles for Service Accounts 

Pass `region`, `accessKeyId` and `secretAccessKey` as empty strings to intialize AWS SDK without credentials, effectively falling back to Service Accounts
