def f(event):
    # event is the JSON-parsed Kafka message value
    print(f"Received message: {event}")
    return {"status": "ok"}
