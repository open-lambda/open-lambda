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
