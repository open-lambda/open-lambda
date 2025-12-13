from flask import Flask, render_template, request

def page_not_found(e):
    return f"{e}, {request.base_url}, {request.url_root}\n", 404

app = Flask(__name__)
app.register_error_handler(404, page_not_found)

@app.route("/home")
def hi():
    return render_template("index.html")

@app.route("/home/button")
def button():
    print("Button endpoint hit!")
    return "You clicked the button!\n"
