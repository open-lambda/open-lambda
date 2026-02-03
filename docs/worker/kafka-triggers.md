# Kafka Triggers

Lambdas can be configured to automatically consume messages from Kafka
topics. When a message arrives, the worker invokes the lambda with the
message payload as the request body.

## Configuration

Add a `kafka` section under `triggers` in your lambda's `ol.yaml`
(see [lambda configuration](lambda-config.md) for the full `ol.yaml`
reference):

```yaml
triggers:
  kafka:
    - bootstrap_servers:
        - "localhost:9092"
      topics:
        - "my-topic"
      auto_offset_reset: "latest" # or "earliest"
```

The consumer group ID is automatically set to `lambda-<name>` based on
the lambda name and cannot be overridden. Because each lambda gets its
own group ID, lambdas consume from Kafka independently of one another.
Even if multiple lambdas subscribe to the same topic, each one receives
its own copy of every message, and offset tracking is maintained
separately per lambda.

## How it works

1. When the worker starts in `lambda` mode, it creates a `KafkaManager`
   alongside the `LambdaServer`.
2. Kafka triggers are registered via the `/kafka/register/<lambda-name>`
   HTTP endpoint (POST to register, DELETE to unregister).
3. For each trigger entry, the manager creates a `LambdaKafkaConsumer`
   backed by a [franz-go](https://github.com/twmb/franz-go) (`kgo`)
   client.
4. Each consumer runs a polling loop that fetches messages with a
   1-second timeout. On receiving a message, it builds a synthetic HTTP
   POST request and invokes the lambda directly through the
   `LambdaManager`.

## Request format

When a Kafka message triggers a lambda, the worker builds a synthetic
HTTP POST request with the Kafka message value as the body and the
following headers:

| Header              | Description                              |
| ------------------- | ---------------------------------------- |
| `Content-Type`      | `application/json`                       |
| `X-Kafka-Topic`     | The topic the message was read from.     |
| `X-Kafka-Partition` | The partition number.                    |
| `X-Kafka-Offset`    | The message offset within the partition. |
| `X-Kafka-Group-Id`  | The consumer group ID.                   |

### Accessing Kafka metadata in your handler

The default handler type (`def f(event)`) only receives the JSON-parsed
request body as a dict. It does **not** have access to HTTP headers,
so the Kafka metadata headers listed above will not be available.

To access Kafka metadata headers, use a **WSGI** or **ASGI** entry
point (see [lambda configuration](lambda-config.md) for how to
configure these).

## Example lambdas

### Simple handler (body only)

The default `f(event)` handler receives the Kafka message body as a
parsed dict, but cannot access headers:

```python
def f(event):
    # event is the JSON-parsed Kafka message value
    print(f"Received: {event}")
    return {"status": "ok"}
```

### WSGI handler (body + headers)

A WSGI handler can access Kafka metadata via the `environ` dict.
HTTP headers are available with an `HTTP_` prefix, uppercased, and
with dashes replaced by underscores:

```python
from flask import Flask, request

app = Flask(__name__)

@app.route("/", methods=["POST"])
def handle():
    topic = request.headers.get("X-Kafka-Topic", "unknown")
    body = request.get_json()

    print(f"Received message from {topic}: {body}")
    return {"status": "ok"}
```

## Management API

The worker exposes an HTTP endpoint for managing Kafka consumers at
runtime:

- **`POST /kafka/register/<lambda-name>`** — Reads the lambda's
  `ol.yaml` config from the registry and starts consumers for all
  configured Kafka triggers. Any existing consumers for that lambda are
  cleaned up first.
- **`DELETE /kafka/register/<lambda-name>`** — Stops and removes all
  Kafka consumers for the given lambda.

## Shutdown

When the worker receives a shutdown signal (SIGTERM/SIGINT), the Kafka
manager is cleaned up before the lambda server. Each consumer's polling
loop is stopped via its stop channel, and the underlying `kgo` client is
closed.
