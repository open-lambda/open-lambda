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

## Quick start

This walkthrough starts a Kafka broker, deploys a lambda with a Kafka
trigger, and publishes a message to verify end-to-end.

### 1. Start a Kafka broker

The easiest way to get a single-node broker is with Docker. The
[apache/kafka](https://hub.docker.com/r/apache/kafka) image bundles
KRaft mode so no separate ZooKeeper container is needed:

```bash
docker run -d --name kafka \
  -p 9092:9092 \
  apache/kafka:latest
```

See the [Apache Kafka quickstart](https://kafka.apache.org/quickstart)
for more details.

### 2. Create a topic

```bash
docker exec kafka \
  /opt/kafka/bin/kafka-topics.sh --create \
    --topic my-topic \
    --bootstrap-server localhost:9092
```

### 3. Write the lambda

Create a directory for the lambda with two files:

**f.py**
```python
def f(event):
    print(f"Received: {event}")
    return {"status": "ok"}
```

**ol.yaml**
```yaml
triggers:
  kafka:
    - bootstrap_servers:
        - "localhost:9092"
      topics:
        - "my-topic"
      auto_offset_reset: "earliest"
```

Upload the lambda to the registry. When a lambda with Kafka triggers is
uploaded, the worker automatically starts consumers for the configured
topics — no extra registration step is needed.

### 4. Publish a test message

```bash
echo '{"hello":"world"}' | docker exec -i kafka \
  /opt/kafka/bin/kafka-console-producer.sh \
    --topic my-topic \
    --bootstrap-server localhost:9092
```

The worker should pick up the message and invoke your lambda. Check the
worker logs to confirm.

## How it works

1. When the worker starts in `lambda` mode, it creates a `KafkaManager`
   alongside the `LambdaServer`.
2. When a lambda with Kafka triggers is uploaded, the boss automatically
   registers its Kafka consumers on the worker. Consumers can also be
   managed manually via the `/kafka/register/<lambda-name>` HTTP
   endpoint (POST to register, DELETE to unregister).
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

Complete working examples are available in the
[examples/](../../examples/) directory:

- [kafka-basic](../../examples/kafka-basic/) — Simple `f(event)` handler
  that processes the Kafka message body.
- [kafka-metadata](../../examples/kafka-metadata/) — Flask WSGI handler
  that accesses Kafka metadata headers (topic, partition, offset, group
  ID) alongside the message body.

### Simple handler (body only)

The default `f(event)` handler receives the Kafka message body as a
parsed dict, but cannot access headers
([full example](../../examples/kafka-basic/)):

```python
def f(event):
    # event is the JSON-parsed Kafka message value
    print(f"Received message: {event}")
    return {"status": "ok"}
```

### WSGI handler (body + headers)

A WSGI handler can access Kafka metadata via the `environ` dict.
HTTP headers are available with an `HTTP_` prefix, uppercased, and
with dashes replaced by underscores
([full example](../../examples/kafka-metadata/)):

```python
from flask import Flask, request

app = Flask(__name__)

@app.route("/", methods=["POST"])
def handle():
    topic = request.headers.get("X-Kafka-Topic", "unknown")
    partition = request.headers.get("X-Kafka-Partition", "unknown")
    offset = request.headers.get("X-Kafka-Offset", "unknown")
    group_id = request.headers.get("X-Kafka-Group-Id", "unknown")

    body = request.get_json()

    print(f"topic={topic} partition={partition} offset={offset} group={group_id}")
    print(f"body={body}")

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
