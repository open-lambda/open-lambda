from flask import Flask, Response

# Intentionally NOT named "app" to test OL_WSGI_ENTRY
my_wsgi_app = Flask("wsgi-entry-test")

@my_wsgi_app.route("/")
def index():
    return Response("Hello from my_wsgi_app!\n", status=200)

@my_wsgi_app.route("/info")
def info():
    return {
        "entry_file": "main.py",
        "entry_point": "my_wsgi_app",
        "message": "This tests OL_WSGI_ENTRY with a non-standard name"
    }
