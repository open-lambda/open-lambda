#!/usr/bin/env python3
"""
Send numbered messages to the 'numbers' Kafka topic.

Usage:
    python produce.py              # send numbers 1..10
    python produce.py 100          # send numbers 1..100
    python produce.py 50 0.5       # send 1..50 with 0.5s delay between each
"""

import json
import sys
import time

from kafka import KafkaProducer

BROKER = "localhost:9092"
TOPIC = "numbers"


def main():
    count = int(sys.argv[1]) if len(sys.argv) > 1 else 10
    delay = float(sys.argv[2]) if len(sys.argv) > 2 else 0.1

    producer = KafkaProducer(
        bootstrap_servers=BROKER,
        value_serializer=lambda v: json.dumps(v).encode("utf-8"),
    )

    expected_sum = 0
    print(f"Sending numbers 1..{count} to topic '{TOPIC}'...")
    for i in range(1, count + 1):
        producer.send(TOPIC, {"number": i})
        expected_sum += i
        print(f"  sent {i}")
        if delay:
            time.sleep(delay)

    producer.flush()
    producer.close()
    print(f"\nDone. Expected sum = {expected_sum}")


if __name__ == "__main__":
    main()
