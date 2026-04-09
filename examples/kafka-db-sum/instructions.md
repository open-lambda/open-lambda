# kafka-db-sum: Testing Instructions

## Prerequisites

- OpenLambda built (`make ol imgs/ol-min`)
- Docker installed

## 1. Start PostgreSQL

```bash
docker run -d --name ol-pg \
  --network host \
  -e POSTGRES_USER=ol \
  -e POSTGRES_PASSWORD=ol \
  -e POSTGRES_DB=ol_demo \
  postgres:16
```

## 2. Start Kafka

```bash
docker run -d --name kafka \
  -p 9092:9092 \
  apache/kafka:latest
```

## 3. Create the `numbers` topic

```bash
docker exec kafka /opt/kafka/bin/kafka-topics.sh --create \
  --topic numbers \
  --bootstrap-server localhost:9092
```

## 4. Initialize and start the OL worker

From the repository root:

```bash
sudo -A ./ol worker init -p ../default-ol -i ol-min
sudo -A ./ol worker up -p ../default-ol
```

Run `worker up` in a separate terminal, or add `-d` for detached mode.
The worker listens on `localhost:5000` by default.

## 5. Install the lambda

From the repository root:

```bash
./ol admin install examples/kafka-db-sum/
```

## 6. Register the Kafka consumer

A standalone worker does not auto-register Kafka triggers on upload.
Register manually:

```bash
curl -X POST localhost:5000/kafka/register/kafka-db-sum
```

## 7. Send test messages

Python producer script (requires `pip install kafka-python`):

```bash
python examples/kafka-db-sum/produce.py 100
```

## 8. Check results

```bash
curl localhost:5000/run/kafka-db-sum/
```

Expected output (sum of 1..100 = 5050):

```json
{ "last_offset": 99, "message_count": 100, "running_sum": 5050 }
```

## 9. Reset and re-run

```bash
curl -X POST localhost:5000/run/kafka-db-sum/reset
```

Then send a fresh batch (step 7) and verify again.

## Configuration

In `ol.yaml`:

| Variable           | Default                                     | Description                                                         |
| ------------------ | ------------------------------------------- | ------------------------------------------------------------------- |
| `DATABASE_URL`     | `postgresql://ol:ol@127.0.0.1:5432/ol_demo` | PostgreSQL connection string                                        |
| `FAIL_PROBABILITY` | `0`                                         | Chance (0.0-1.0) of simulated failure. Use `0.3` to test seek-back. |

## Cleanup

```bash
sudo -A ./ol worker down -p default-ol
docker rm -f kafka ol-pg
```
