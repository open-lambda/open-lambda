import time

def f(event):
    time.sleep(int(event))
    return f"slept {event} second\n"