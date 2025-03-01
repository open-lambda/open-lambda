import requests

def f(event):
        url = "https://c3ccc56e-4638-48bb-8629-58af653dd127-00-jezlxaiubmz7.janeway.replit.dev/"
        response = requests.get(url)
        print("Status code:", response.status_code)
        print("Respons1e body:", response.text)
        print(event)
