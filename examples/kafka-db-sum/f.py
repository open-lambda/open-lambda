from flask import Flask, request, make_response, jsonify
import os
import random
import psycopg2

app = Flask(__name__)

DATABASE_URL = os.environ.get(
    "DATABASE_URL", "postgresql://ol:ol@localhost:5432/ol_demo"
)
# Probability (0.0–1.0) that a transaction will fail between UPDATE and COMMIT.
# Set to 0 for normal operation; raise to stress-test seek-back recovery.
FAIL_PROBABILITY = float(os.environ.get("FAIL_PROBABILITY", "0"))
_db_initialized = False


def get_db():
    return psycopg2.connect(DATABASE_URL)


def ensure_db():
    """Create the running_sum table if it doesn't exist (runs once per sandbox)."""
    global _db_initialized
    if _db_initialized:
        return
    conn = get_db()
    try:
        with conn.cursor() as cur:
            cur.execute(
                """
                CREATE TABLE IF NOT EXISTS running_sum (
                    id INTEGER PRIMARY KEY DEFAULT 1,
                    total BIGINT NOT NULL DEFAULT 0,
                    last_offset BIGINT NOT NULL DEFAULT -1,
                    message_count BIGINT NOT NULL DEFAULT 0,
                    CHECK (id = 1)
                )
                """
            )
            cur.execute(
                """
                INSERT INTO running_sum (id, total, last_offset, message_count)
                VALUES (1, 0, -1, 0)
                ON CONFLICT (id) DO NOTHING
                """
            )
        conn.commit()
        _db_initialized = True
    finally:
        conn.close()


@app.route("/reset", methods=["POST"])
def reset():
    """Reset running_sum to zero so the demo can be re-run cleanly."""
    ensure_db()
    conn = get_db()
    try:
        with conn.cursor() as cur:
            cur.execute(
                "UPDATE running_sum SET total = 0, last_offset = -1, "
                "message_count = 0 WHERE id = 1"
            )
        conn.commit()
        return jsonify({"status": "reset"})
    finally:
        conn.close()


@app.route("/", methods=["GET", "POST"])
def handle():
    ensure_db()

    # GET — return current state (useful for checking progress via HTTP)
    if request.method == "GET":
        conn = get_db()
        try:
            with conn.cursor() as cur:
                cur.execute(
                    "SELECT total, last_offset, message_count "
                    "FROM running_sum WHERE id = 1"
                )
                row = cur.fetchone()
                if row:
                    return jsonify(
                        {
                            "running_sum": row[0],
                            "last_offset": row[1],
                            "message_count": row[2],
                        }
                    )
                return jsonify(
                    {"running_sum": 0, "last_offset": -1, "message_count": 0}
                )
        finally:
            conn.close()

    # POST — process a Kafka message containing a number
    offset = int(request.headers.get("X-Kafka-Offset", "-1"))
    topic = request.headers.get("X-Kafka-Topic", "unknown")
    partition = request.headers.get("X-Kafka-Partition", "unknown")

    body = request.get_json(silent=True)

    # Accept {"number": N}, bare int, or string
    if isinstance(body, dict):
        number = body.get("number", 0)
    elif isinstance(body, (int, float)):
        number = body
    else:
        try:
            number = int(body)
        except (TypeError, ValueError):
            number = 0

    conn = None
    try:
        conn = get_db()
        with conn.cursor() as cur:
            # Lock the row and read last processed offset
            cur.execute(
                "SELECT last_offset FROM running_sum WHERE id = 1 FOR UPDATE"
            )
            row = cur.fetchone()
            last_offset = row[0] if row else -1

            # Idempotency: skip if this offset was already processed.
            # This prevents double-counting after a seek-back replays
            # messages that were already committed.
            if offset <= last_offset:
                conn.rollback()
                print(f"[skip] offset={offset} already processed (last={last_offset})")
                return jsonify(
                    {
                        "status": "skipped",
                        "reason": "already processed",
                        "offset": offset,
                        "last_offset": last_offset,
                    }
                )

            # Atomically add number to running sum and advance the offset
            cur.execute(
                """
                UPDATE running_sum
                SET total = total + %s,
                    last_offset = %s,
                    message_count = message_count + 1
                WHERE id = 1
                """,
                (number, offset),
            )

            # --- Fault injection ------------------------------------------------
            # Simulate a crash between UPDATE and COMMIT.  The UPDATE is in the
            # transaction buffer but NOT committed, so the exception triggers a
            # rollback — exactly the scenario the seek-back mechanism is designed
            # to recover from.
            if FAIL_PROBABILITY > 0 and random.random() < FAIL_PROBABILITY:
                raise Exception(
                    f"Simulated DB failure at offset {offset} "
                    f"(FAIL_PROBABILITY={FAIL_PROBABILITY})"
                )
            # --------------------------------------------------------------------

            conn.commit()

            # Read back the new state for the response
            cur.execute(
                "SELECT total, last_offset, message_count "
                "FROM running_sum WHERE id = 1"
            )
            total, last_off, count = cur.fetchone()

        print(f"[ok] offset={offset} number={number} sum={total} count={count}")
        return jsonify(
            {
                "status": "ok",
                "offset": offset,
                "number_added": number,
                "running_sum": total,
                "message_count": count,
            }
        )

    except Exception as e:
        print(f"[error] offset={offset} error={e}")
        if conn:
            try:
                conn.rollback()
            except Exception:
                pass

        # Tell OL's Kafka consumer to seek back to this offset and retry.
        # The consumer's LRU cache will serve the replay without re-fetching
        # from Kafka, and the idempotency check above prevents double-counting
        # for any offsets that were already committed before the failure.
        resp = make_response(
            jsonify({"status": "error", "offset": offset, "error": str(e)}), 500
        )
        resp.headers["X-Kafka-Seek-Offset"] = str(offset)
        return resp

    finally:
        if conn:
            try:
                conn.close()
            except Exception:
                pass
