# CLAUDE.md — Developer & AI Agent Guide for pubsublib

## Repository Purpose

`pubsublib` is a Go library (`github.com/Orange-Health/pubsublib`) that provides a unified publish/subscribe abstraction over AWS SNS/SQS and Redis. Consuming services import the library and interact through a single `Provider` interface without coupling business logic to a specific broker.

---

## Directory Map

```
pubsublib/
├── pubsub.go               # Core types: Client, Provider, Middleware, MessageHandler, SetClient
├── public.go               # Client.Publish implementation + chainPublisherMiddleware
├── noop.go                 # NoopProvider — default stub (no-ops all calls)
│
├── provider/
│   ├── aws/
│   │   └── aws.go          # AWSPubSubAdapter: SNS publish, SQS poll, Redis overflow, compression
│   └── redis/
│       └── redis.go        # RedisPubSubAdapter: native Redis Pub/Sub
│
├── infrastructure/
│   └── redis-client.go     # Singleton RedisDatabase (go-redis/v8); key prefix PUBSUB:
│
├── helper/
│   ├── codec.go            # Gzip compress/decompress + Base64 encode/decode
│   └── json_formatting.go  # Recursive camelCase → snake_case map key converter
│
├── docs/
│   └── architecture/       # Architecture documentation (Mermaid diagrams)
│       ├── overview.md
│       ├── service-layer.md
│       ├── integrations.md
│       ├── configuration.md
│       ├── observability.md
│       └── branching.md
│
├── .github/
│   ├── CODEOWNERS
│   └── PULL_REQUEST_TEMPLATE.md
│
├── CLAUDE.md               # This file
├── AGENTS.md               # Agent-focused quick reference
└── README.md
```

---

## Layering Model

```
Consumer Service
     │
     ▼
pubsub.Client  (+ Middleware chain)
     │
     ▼
Provider interface
     │
     ├── AWSPubSubAdapter   (production)
     │       ├── AWS SNS    (publish)
     │       ├── AWS SQS    (consume)
     │       └── RedisDatabase (overflow for payloads > 200 KB)
     │
     ├── RedisPubSubAdapter (dev / intra-service)
     │       └── Redis Pub/Sub
     │
     └── NoopProvider       (test / default)
```

---

## Key Conventions

### Message attributes (required on every publish)

| Key | Description |
|---|---|
| `source` | Originating service identifier |
| `contains` | Entity / resource type in the payload |
| `event_type` | Snake-case semantic event name (AWS adapter) |
| `eventType` | Camel-case semantic event name (Redis adapter) |

`trace_id` is auto-generated (UUID v4) if not provided.

### Body serialisation
- Messages are JSON-marshalled before publishing.
- Map keys are automatically converted to `snake_case` by `helper.ConvertBodyToSnakeCase`.

### Large-message overflow
- Bodies exceeding **200 KB** are stored in Redis under `PUBSUB:<uuid>` with a **2-minute TTL**.
- The SNS body is replaced with a pointer string; the `redis_key` attribute carries the UUID.
- Consumers transparently hydrate the body from Redis on `PollMessages`.

### Compression
- Set `PUBSUBLIB_COMPRESSION_ENABLED=true` to enable gzip + base64 encoding.
- Or call `adapter.SetCompressionEnabled(true)` programmatically.
- Compressed messages carry `compress="true"` in message attributes.

---

## Adding a New Feature

### Adding a new Provider

1. Create `provider/<name>/<name>.go`.
2. Implement the `pubsub.Provider` interface:
   ```go
   type Provider interface {
       Publish(topicARN string, message interface{}, messageAttributes map[string]interface{}) error
       PollMessages(queueURL string, handler MessageHandler) error
   }
   ```
3. Export a `New<Name>PubSubAdapter(...)` constructor.
4. Update `docs/architecture/integrations.md` and `docs/architecture/overview.md`.

### Adding a new Middleware

1. Implement `pubsub.Middleware`:
   ```go
   type Middleware interface {
       PublisherMsgInterceptor(serviceName string, next PublishHandler) PublishHandler
   }
   ```
2. Register it in `Client.Middleware` at construction time.
3. Document any new observability signals in `docs/architecture/observability.md`.

### Modifying message attribute handling

- Required attribute validation lives in `provider/aws/aws.go` (`Publish`) and `provider/redis/redis.go` (`Publish`).
- Both providers must be updated consistently when adding new required attributes.

---

## Branching Rules

| Branch | Usage |
|---|---|
| `main` | Production only; hotfixes directly |
| `dev` | Integration; all feature PRs target here |
| `release` | Cut from `dev`; RC tags applied here |
| `feature/<JIRAID>-<name>` | Feature work; branches from and merges into `dev` |
| `hotfix/<JIRAID>-<desc>` | Urgent fixes; branches from `main`; merges into `main` + `dev` + `release` |

### Versioning
- Production: `YY.MMDD.PATCHCOUNT` (e.g. `v26.0401.1`)
- Release candidate: `RC-YY.MMDD.BUILDCOUNT`
- Staging: `vstag-b<NN>`, `vstagsandbox-b<NN>`

---

## Running Tests

Since the library has no `go.mod` at root (it is a module published from the repo root via GitHub), tests are run from each package directory:

```bash
go test ./...
```

Use `NoopProvider` or pass a mock `Provider` implementation via `pubsub.SetClient` in tests — no real AWS or Redis connection required.

---

## Common Gotchas

1. **`AWSPubSubAdapter` does not implement `pubsub.Provider`** — its `Publish` and `PollMessages` signatures differ (extra FIFO parameters, typed SQS handler). Use it directly; do not assign it to `pubsub.Client.Provider`.
2. **`NewRedisDatabase` is a singleton** — passing different parameters on a second call has no effect; the original connection pool is returned.
3. **`redis_key` typo in Redis adapter** — `messageAttributs` (missing 'e') is intentional for backwards compatibility; do not "fix" it without a migration plan.
4. **Compression is checked at construction time** — changing `PUBSUBLIB_COMPRESSION_ENABLED` after the adapter is created has no effect unless `SetCompressionEnabled` is called explicitly.
