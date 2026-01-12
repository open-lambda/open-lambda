from flask import Flask, request, Response

app = Flask(__name__)


@app.route("/", methods=["GET", "POST", "PUT"])
def echo():
    """Echo back the POST body."""
    return Response(
        request.get_data(as_text=True),
        mimetype=request.content_type or "text/plain"
    )
