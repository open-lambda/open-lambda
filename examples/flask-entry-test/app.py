from flask import Flask, request, Response

app = Flask("flask-entry-test")

@app.route("/")
def index():
    return Response("Hello from app.py!\n", status=200)

@app.route("/info")
def info():
    return {
        "entry_file": "app.py",
        "message": "This function uses OL_ENTRY_FILE to specify app.py as the entry point"
    }
