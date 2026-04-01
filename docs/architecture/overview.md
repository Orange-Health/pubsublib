# pubsublib — Architecture Overview

`pubsublib` is a Go library that provides a unified publish/subscribe abstraction over multiple message-transport backends. Consuming services import the library, initialise the adapter of their choice, and interact through a single, stable API surface — without coupling their business logic to a specific broker.

---

## High-Level Component Map

```mermaid
graph TD
    subgraph Consumers["Consumer Services"]
        SVC[Your Go Service]
    end

    subgraph Core["pubsublib Core (package pubsub)"]
        CLIENT[Client]
        MW[Middleware Chain]
        IFACE[Provider Interface]
    end

    subgraph Providers["Transport Providers"]
        AWS[provider/aws — AWSPubSubAdapter]
        REDIS[provider/redis — RedisPubSubAdapter]
        NOOP[NoopProvider]
    end

    subgraph Infrastructure["Infrastructure Helpers"]
        RDBMS[infrastructure/RedisDatabase\nOverflow Store]
        HELPER_CODEC[helper/codec\nGzip + Base64]
        HELPER_JSON[helper/json_formatting\nSnake-case Converter]
    end

    subgraph External["External Services"]
        SNS[AWS SNS]
        SQS[AWS SQS]
        RDB[Redis]
    end

    SVC -->|pubsub.Client.Publish| CLIENT
    CLIENT --> MW
    MW --> IFACE
    IFACE --> AWS
    IFACE --> REDIS
    IFACE --> NOOP

    AWS --> HELPER_CODEC
    AWS --> HELPER_JSON
    AWS --> RDBMS
    AWS --> SNS
    AWS --> SQS

    RDBMS --> RDB
    REDIS --> RDB
```

---

## Core Abstractions

| Abstraction | Location | Purpose |
|---|---|---|
| `Provider` | `pubsub.go` | Interface that every transport adapter implements (`Publish`, `PollMessages`) |
| `Client` | `pubsub.go` | Holds a `Provider` reference and an ordered `Middleware` slice; exposes `Publish` |
| `Middleware` | `pubsub.go` | Interceptor interface; allows cross-cutting concerns (logging, tracing, retry) on publish |
| `MessageHandler` | `pubsub.go` | `func(string) error` callback used by the base provider contract |
| `NoopProvider` | `noop.go` | No-op implementation; default out-of-the-box so tests never need real brokers |

---

## Publish Flow

```mermaid
sequenceDiagram
    participant SVC as Consumer Service
    participant CLI as pubsub.Client
    participant MWC as Middleware Chain
    participant PRV as Provider (AWS / Redis)
    participant SNS as AWS SNS
    participant RDB as Redis (overflow)

    SVC->>CLI: Publish(topicARN, message, attrs)
    CLI->>MWC: chainPublisherMiddleware(...)
    MWC->>PRV: PublishHandler(topicARN, message, attrs)
    alt message > 200 KB
        PRV->>RDB: Set(uuid, body, 2 min TTL)
        PRV-->>PRV: replace body with redis pointer
    end
    alt compression enabled
        PRV-->>PRV: Gzip + Base64 encode
        PRV-->>PRV: set attrs["compress"]="true"
    end
    PRV->>SNS: sns.Publish(...)
    SNS-->>PRV: MessageId
    PRV-->>CLI: nil / error
    CLI-->>SVC: nil / error
```

---

## Poll / Consume Flow

```mermaid
sequenceDiagram
    participant WORKER as Consumer Worker
    participant ADP as AWSPubSubAdapter
    participant SQS as AWS SQS
    participant RDB as Redis (overflow)
    participant HDL as MessageHandler

    WORKER->>ADP: PollMessages(queueURL, handler)
    ADP->>SQS: ReceiveMessage (batch ≤10)
    SQS-->>ADP: []Message
    loop for each message
        ADP->>ADP: MD5 integrity check
        opt redis_key attribute present
            ADP->>RDB: Get(redis_key)
            RDB-->>ADP: full body
        end
        opt SNS envelope detected
            ADP->>ADP: unwrap Message field
        end
        opt compress attribute = true
            ADP->>ADP: Base64Decode + Gunzip
        end
        ADP->>HDL: handler(message)
        ADP->>SQS: DeleteMessage
    end
```

---

## Package Layout

```
pubsublib/
├── pubsub.go               # Core types: Client, Provider, Middleware, MessageHandler
├── public.go               # Client.Publish + middleware chaining
├── noop.go                 # NoopProvider (default / test stub)
│
├── provider/
│   ├── aws/
│   │   └── aws.go          # AWSPubSubAdapter: SNS publish, SQS poll, Redis overflow
│   └── redis/
│       └── redis.go        # RedisPubSubAdapter: native Redis Pub/Sub
│
├── infrastructure/
│   └── redis-client.go     # Singleton RedisDatabase (go-redis/v8); keys prefixed PUBSUB:
│
├── helper/
│   ├── codec.go            # Gzip compress/decompress; Base64 encode/decode
│   └── json_formatting.go  # Recursive camelCase → snake_case map conversion
│
└── .github/
    ├── CODEOWNERS
    └── PULL_REQUEST_TEMPLATE.md
```

---

## Design Principles

1. **Provider-agnostic interface** — swap AWS for Redis (or a future Kafka adapter) with zero changes to business logic.
2. **Composable middleware** — interceptors stack in reverse-registration order, enabling clean separation of cross-cutting concerns without modifying the core path.
3. **Large-message transparency** — payloads exceeding the SNS/SQS 256 KB limit are automatically offloaded to Redis; consumers transparently retrieve and hydrate the full body.
4. **Optional compression** — gzip + base64 is opt-in via an environment variable, keeping the hot path lean for small payloads.
5. **Zero-dependency default** — `NoopProvider` is registered as the default client so imported packages never panic during tests or cold starts.
