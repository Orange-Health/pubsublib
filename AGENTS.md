# AGENTS.md — AI Agent Quick Reference for pubsublib

## What this repo is

A Go library (`github.com/Orange-Health/pubsublib`) that abstracts AWS SNS/SQS and Redis Pub/Sub behind a common `Provider` interface with optional middleware, gzip compression, and large-message Redis overflow.

---

## Key Files to Read First

| File | Why |
|---|---|
| `pubsub.go` | Core types: `Client`, `Provider`, `Middleware`, `MessageHandler` |
| `public.go` | `Client.Publish` + `chainPublisherMiddleware` |
| `provider/aws/aws.go` | Primary production adapter (SNS publish, SQS poll, Redis overflow) |
| `provider/redis/redis.go` | Redis Pub/Sub adapter |
| `infrastructure/redis-client.go` | Singleton Redis client for overflow |
| `helper/codec.go` | Gzip + Base64 utilities |
| `helper/json_formatting.go` | camelCase → snake_case converter |

---

## Architecture in One Paragraph

`pubsub.Client` holds a `Provider` and an ordered `Middleware` slice. When `Client.Publish` is called, middleware is chained in reverse order and the final call is delegated to `Provider.Publish`. `AWSPubSubAdapter` serialises the message to JSON, optionally compresses it, offloads payloads larger than 200 KB to Redis under a UUID key with a 2-minute TTL, validates required message attributes (`source`, `contains`, `event_type`), and calls `sns.Publish`. Consumers call `AWSPubSubAdapter.PollMessages`, which batches up to 10 SQS messages, verifies MD5 integrity, hydrates any Redis-offloaded bodies, decompresses if needed, calls the handler, and deletes the message from SQS.

---

## Interface Contracts

### `Provider`
```go
type Provider interface {
    Publish(topicARN string, message interface{}, messageAttributes map[string]interface{}) error
    PollMessages(queueURL string, handler MessageHandler) error
}
```

### `Middleware`
```go
type Middleware interface {
    PublisherMsgInterceptor(serviceName string, next PublishHandler) PublishHandler
}
```

### `MessageHandler`
```go
type MessageHandler func(message string) error
```

---

## Required Message Attributes

Every `Publish` call must include:

```go
map[string]interface{}{
    "source":     "service-name",
    "contains":   "entity-type",
    "event_type": "event.name",   // AWS adapter
    // "eventType": "eventName",  // Redis adapter
}
```

`trace_id` is auto-injected as a UUID v4 if absent.

---

## Environment Variables

| Variable | Effect |
|---|---|
| `PUBSUBLIB_COMPRESSION_ENABLED=true` | Enable gzip+base64 compression on publish |

---

## Safe Defaults

- `NoopProvider` is the default — safe to import the library without any configuration.
- `pubsub.SetClient(&pubsub.Client{Provider: myMock})` to inject a mock in tests.

---

## Branching Conventions (for PRs and commits)

| Scenario | Branch pattern |
|---|---|
| New feature | `feature/<JIRAID>-<kebab-description>` from `dev` → PR to `dev` |
| Hotfix | `hotfix/<JIRAID>-<kebab-description>` from `main` → PR to `main` + `dev` + `release` |
| Release | `release` cut from `dev`; tagged `RC-YY.MMDD.BUILDCOUNT` during stabilisation |
| Production tag | `vYY.MMDD.PATCHCOUNT` applied to `main` after release merge |

---

## What NOT to do

- Do **not** assign `AWSPubSubAdapter` to `pubsub.Client.Provider` — signatures are incompatible.
- Do **not** call `NewRedisDatabase` multiple times expecting different pools — it is a singleton.
- Do **not** rename `messageAttributs` in `provider/redis/redis.go` without a consumer migration — it is a backwards-compatible typo.
- Do **not** delete or move existing version tags — they are immutable module references.

---

## Documentation Index

| Document | Topic |
|---|---|
| `docs/architecture/overview.md` | Component map, publish/consume flows, package layout |
| `docs/architecture/service-layer.md` | Layer-by-layer breakdown with code references |
| `docs/architecture/integrations.md` | AWS SNS, SQS, Redis integration details and contracts |
| `docs/architecture/configuration.md` | All constructor params, env vars, and compile-time constants |
| `docs/architecture/observability.md` | Built-in tracing, logging, and instrumentation patterns |
| `docs/architecture/branching.md` | Branch model, naming conventions, versioning scheme |
