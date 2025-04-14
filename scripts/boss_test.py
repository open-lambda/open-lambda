import os, sys
import json
import time
from subprocess import run
import requests
import tarfile
import tempfile

api_key = None
boss_port = 5000

def read_json(path):
    with open(path, encoding="utf-8") as f:
        return json.load(f)

def write_json(path, data):
    with open(path, "w", encoding="utf-8") as f:
        return json.dump(data, f, indent=2)

def boss_get(resource, check=True):
    url = f"http://localhost:{boss_port}/{resource}"
    resp = requests.get(url)
    if check:
        resp.raise_for_status()
    return resp.text

def boss_post(resource, data, check=True):
    url = f"http://localhost:{boss_port}/{resource}"
    resp = requests.post(url, headers={"api_key": api_key}, data=data)
    if check:
        resp.raise_for_status()
    return resp

def boss_invoke(lambda_name, data, check=True):
    url = f"http://localhost:{boss_port}/run/{lambda_name}"
    resp = requests.post(url, data=data)
    if check:
        resp.raise_for_status()
    return resp

def tester(platform):
    global api_key, boss_port
    print(f"Testing {platform}")

    # PART 0: clear existing config
    if os.path.exists("boss.json"):
        run(["rm", "boss.json"]).check_returncode()

    # PART 1: config and launch
    run(["./ol", "boss", "--detach"]).check_returncode()
    assert os.path.exists("boss.json")

    config = read_json("boss.json")
    assert "platform" in config
    assert "scaling" in config
    config["platform"] = platform
    config["scaling"] = "manual"
    write_json("boss.json", config)

    assert "api_key" in config
    assert "boss_port" in config
    api_key = config["api_key"]
    boss_port = config["boss_port"]

    time.sleep(1)  # give boss time to boot

    # PART 2: scaling
    status = json.loads(boss_get("status"))
    assert status["state"]["running"] == 0

    boss_post("scaling/worker_count", "1")
    status = json.loads(boss_get("status"))
    assert status["state"]["starting"] == 1

    # wait until worker is ready
    t0 = time.time()
    while time.time() - t0 < 180:
        time.sleep(1)
        status = json.loads(boss_get("status"))
        if status["state"]["running"] == 1:
            break
    else:
        raise RuntimeError("Timeout waiting for worker to be ready")

    # PART 3: upload lambda
    lambda1_name = "hi"
    code = ["def f(event):", "\treturn 'hello'"]
    tar_path = create_lambda_tar(lambda1_name, code)

    with open(tar_path, "rb") as f:
        files = {"file": (f"{lambda1_name}.tar.gz", f, "application/gzip")}
        resp = requests.post(f"http://localhost:{boss_port}/registry/{lambda1_name}", files=files)
        resp.raise_for_status()

    os.remove(tar_path)

    # PART 4: invoke lambda
    result = boss_invoke(f"run/{lambda1_name}", None).json()
    assert result == 'hello'

    # PART 5: test after scaling down
    boss_post("scaling/worker_count", "0")
    time.sleep(1)
    status = json.loads(boss_get("status"))
    assert status["state"]["running"] == 0

    resp = boss_invoke(f"run/{lambda1_name}", None, check=False)
    assert resp.status_code != 200
    
    
def create_lambda_tar(lambda_name, code_lines):
    with tempfile.NamedTemporaryFile(delete=False, mode="w", suffix=".py") as code_file:
        code_file.write("\n".join(code_lines))
        code_path = code_file.name

    with tempfile.NamedTemporaryFile(delete=False, mode="w", suffix=".yaml") as ol_file:
        ol_file.write("triggers:\n  http:\n    - method: \"*\"\n")
        ol_path = ol_file.name

    temp_tar_path = tempfile.NamedTemporaryFile(delete=False, suffix=".tar.gz").name
    with tarfile.open(temp_tar_path, "w:gz") as tar:
        tar.add(code_path, arcname="f.py")
        tar.add(ol_path, arcname="ol.yaml")

    os.remove(code_path)
    os.remove(ol_path)

    return temp_tar_path

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 boss-test.py (aws|azure|gcp) [platform2, ...]")

    for platform in sys.argv[1:]:
        tester(platform)


if __name__ == "__main__":
    main()
