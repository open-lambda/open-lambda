from flask import Flask, request, Response

def page_not_found(e):
  return f"{e}, {request.base_url}, {request.url_root}\n", 404

app = Flask("hi")
app.register_error_handler(404, page_not_found)

# TODO: modify wrappers so "/" is the root
@app.route("/run/lambda-config-test", methods=["GET", "POST", "PUT", "DELETE"])
def hi():
  print("in hi() of lambda-config-test/f.py")
  teapot = 418 # I'm a teapot (https://developer.mozilla.org/en-US/docs/Web/HTTP/Status/418)
  return Response("hi\n", status=teapot, headers={"A":"B"})
