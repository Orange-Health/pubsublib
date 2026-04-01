# Configuration

This document lists every configuration surface exposed by `pubsublib` — constructor parameters, environment variables, and runtime toggles.

---

## AWS Adapter — Constructor Parameters

`provider/aws.NewAWSPubSubAdapter` accepts all configuration at construction time:

| Parameter | Type | Required | Description |
|---|---|---|---|
| `region` | `string` | Yes | AWS region (e.g. `ap-south-1`) |
| `accessKeyId` | `string` | No* | AWS access key ID. Pass `""` to use IRSA / instance profile |
| `secretAccessKey` | `string` | No* | AWS secret access key. Pass `""` to use IRSA / instance profile |
| `snsEndpoint` | `string` | No | Override SNS endpoint URL (e.g. `http://localhost:4566` for LocalStack). Pass `""` for the default AWS endpoint |
| `redisAddress` | `string` | Yes | Redis host:port for the overflow store (e.g. `localhost:6379`) |
| `redisPassword` | `string` | No | Redis AUTH password; pass `""` for no auth |
| `redisDB` | `int` | Yes | Redis logical database index |
| `redisPoolSize` | `int` | Yes | Maximum number of Redis connections in the pool |
| `redisMinIdleConn` | `int` | Yes | Minimum number of idle Redis connections kept alive |

\* Pass empty strings for both `accessKeyId` and `secretAccessKey` to enable **IRSA / IAM Role** credential chain.

### Example — static credentials

```go
adapter, err := aws.NewAWSPubSubAdapter(
    "ap-south-1",
    os.Getenv("AWS_ACCESS_KEY_ID"),
    os.Getenv("AWS_SECRET_ACCESS_KEY"),
    "",           // use default SNS endpoint
    "redis:6379",
    "",           // no redis password
    0, 10, 2,
)
```

### Example — IRSA (Kubernetes service accounts)

```go
adapter, err := aws.NewAWSPubSubAdapter(
    "ap-south-1",
    "", "",       // empty → SDK uses IRSA / instance metadata
    "",
    "redis:6379",
    "", 0, 10, 2,
)
```

---

## Environment Variables

| Variable | Values | Default | Description |
|---|---|---|---|
| `PUBSUBLIB_COMPRESSION_ENABLED` | `true`, `1` | disabled | Enable gzip + base64 compression of message bodies before publishing. Checked once at `NewAWSPubSubAdapter` construction time. |

### Setting compression at runtime

Compression can also be toggled programmatically after construction:

```go
adapter.SetCompressionEnabled(true)  // enable
adapter.SetCompressionEnabled(false) // disable
```

---

## Redis Adapter — Constructor Parameters

`provider/redis.NewRedisPubSubAdapter` takes a single address parameter:

| Parameter | Type | Description |
|---|---|---|
| `addr` | `string` | Redis address in `host:port` format |

---

## Redis Infrastructure — Singleton Configuration

`infrastructure.NewRedisDatabase` is a singleton factory:

| Parameter | Type | Description |
|---|---|---|
| `address` | `string` | Redis `host:port` |
| `password` | `string` | Redis AUTH password (`""` for none) |
| `db` | `int` | Logical DB index |
| `poolSize` | `int` | Max active connections |
| `minIdleConn` | `int` | Min idle connections |

Once initialised, subsequent calls return the existing instance regardless of parameters passed.

---

## Message Attribute Requirements

The following keys must be present in `messageAttributes` for every publish call:

| Key | Provider | Description |
|---|---|---|
| `source` | AWS + Redis | Service name or identifier originating the event |
| `contains` | AWS + Redis | Entity / resource type the event carries |
| `event_type` | AWS | Snake-case semantic event name |
| `eventType` | Redis | Camel-case semantic event name (Redis adapter convention) |
| `trace_id` | AWS | Distributed trace ID; auto-populated with UUID v4 if missing |

---

## Large-Message Overflow Thresholds

| Threshold | Value | Behaviour |
|---|---|---|
| Body size limit | 200 KB (`200 * 1024` bytes) | Payload stored in Redis; body replaced with pointer string |
| Redis key TTL | 2 minutes | Keys expire automatically if not consumed |

---

## Compile-Time / Package Defaults

| Constant | Value | Location | Description |
|---|---|---|---|
| `keyPrefix` | `"PUBSUB"` | `infrastructure/redis-client.go` | Prefix for all Redis keys |
| Default provider | `NoopProvider{}` | `pubsub.go` | Active when no adapter is explicitly set |
| SQS visibility timeout | `5 s` | `provider/aws/aws.go` | Per-message processing window |
| SQS max messages | `10` | `provider/aws/aws.go` | Batch size per `ReceiveMessage` call |
