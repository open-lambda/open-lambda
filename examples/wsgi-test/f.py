from flask import Flask, request, Response

def page_not_found(e):
    return f"404 Not Found: {request.method} {request.path}", 404

app = Flask("WSGI")
app.register_error_handler(404, page_not_found)

@app.route("/home")
def hi():
    return Response(f"Request path: {request.path}\n", status=200)

