import os
import sys
import json
import time
import tarfile
import tempfile
import subprocess
from subprocess import run

import requests

# Globals for API interaction
api_key = None
boss_port = 5000

### ------------------ Utility Functions ------------------ ###

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
    return resp.text.strip()

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

### ------------------ Setup Functions ------------------ ###

def clear_config():
    if os.path.exists("boss.json"):
        run(["rm", "boss.json"]).check_returncode()

def launch_boss(platform):
    print(f"[BOOT] Launching boss on platform '{platform}'...")
    global api_key, boss_port
    run(["./ol", "boss", "--detach"]).check_returncode()
    assert os.path.exists("boss.json")

    config = read_json("boss.json")
    config["platform"] = platform
    config["scaling"] = "manual"
    write_json("boss.json", config)

    api_key = config["api_key"]
    boss_port = config["boss_port"]
    time.sleep(1)  # Give boss time to boot
    print("[BOOT] Boss launched and config written.\n")

def scale_workers(count):
    print(f"[SCALE] Scaling to {count} worker(s)...")
    boss_post("scaling/worker_count", str(count))

def wait_for_workers(expected_running, timeout=180):
    print(f"[WAIT] Waiting for {expected_running} running worker(s)...\n")
    t0 = time.time()
    while time.time() - t0 < timeout:
        time.sleep(1)
        status = json.loads(boss_get("status"))
        if status["state"]["running"] == expected_running:
            return
    raise RuntimeError(
        f"Timeout waiting for {expected_running} workers to be running"
    )

### ------------------ Lambda Operations ------------------ ###

def create_lambda_tar(code_lines):
    print("[BUILD] Creating lambda tarball...")
    # Create temp files for f.py and ol.yaml
    with tempfile.NamedTemporaryFile(delete=False, mode="w", suffix=".py") as code_file:
        code_file.write("\n".join(code_lines))
        code_path = code_file.name

    with tempfile.NamedTemporaryFile(delete=False, mode="w", suffix=".yaml") as ol_file:
        ol_file.write("triggers:\n  http:\n    - method: \"POST\"\n")
        ol_path = ol_file.name

    # Create the tar.gz file path
    temp_tar_path = tempfile.mktemp(suffix=".tar.gz")

    # Add both files at the top level in the tarball
    with tarfile.open(temp_tar_path, "w:gz") as tar:
        tar.add(code_path, arcname="f.py")
        tar.add(ol_path, arcname="ol.yaml")
    # Clean up the temporary source files
    os.remove(code_path)
    os.remove(ol_path)

    return temp_tar_path


def upload_lambda(lambda_name, code_lines):
    print(f"[UPLOAD] Uploading lambda '{lambda_name}'...")
    tar_path = create_lambda_tar(code_lines)
    with open(tar_path, "rb") as f:
        url = f"http://localhost:{boss_port}/registry/{lambda_name}"
        headers = {"Content-Type": "application/octet-stream"}
        resp = requests.post(url, data=f, headers=headers)
        resp.raise_for_status()
    os.remove(tar_path)
    print(f"[UPLOAD] Lambda '{lambda_name}' uploaded.\n")

def invoke_lambda(lambda_name, check=True):
    print(f"[INVOKE] Invoking lambda '{lambda_name}'...\n")
    resp = boss_invoke(lambda_name, None, check=check)
    return resp.json() if check else resp

def verify_lambda_config(lambda_name):
    print(f"[VERIFY] Verifying config for lambda '{lambda_name}'...")
    resp = boss_get(f"registry/{lambda_name}/config")
    actual_config = json.loads(resp)
    expected_config = {
        "Triggers": {
            "HTTP": [{"Method": "POST"}],
            "Cron": None,
            "Kafka": None,
        }
    }
    assert actual_config == expected_config, (
        f"Lambda config mismatch!\nExpected: {expected_config}\nActual: {actual_config}"
    )
    print("[VERIFY] Config verified successfully.\n")

def shutdown_and_check(lambda_name):
    print(f"[SHUTDOWN] Shutting down workers for lambda '{lambda_name}'...")
    scale_workers(0)
    time.sleep(1)
    status = json.loads(boss_get("status"))
    assert status["state"]["running"] == 0, (
        f"Expected 0 running workers, got: {status['state']['running']}"
    )
    print("[SHUTDOWN] Workers successfully shut down.\n")

def delete_lambda_and_verify(lambda_name):
    print(f"[DELETE] Deleting lambda '{lambda_name}'...")
    url = f"http://localhost:{boss_port}/registry/{lambda_name}"
    resp = requests.delete(url, headers={"api_key": api_key})
    resp.raise_for_status()

    list_resp = boss_get("registry")
    lambda_list = json.loads(list_resp)
    assert lambda_name not in lambda_list, (
        f"Deleted lambda '{lambda_name}' still listed: {lambda_list}"
    )
    print(f"[DELETE] Lambda '{lambda_name}' deleted and verified.\n")

def shutdown_boss():
    print("[SHUTDOWN] Shutting down Boss...")
    try:
        resp = requests.post(f"http://localhost:{boss_port}/shutdown")
        if resp.status_code == 200:
            print("[SHUTDOWN] Boss shutdown requested successfully.\n")
        else:
            print(f"[SHUTDOWN] Unexpected response code: {resp.status_code}")
    except requests.RequestException as e:
        print(f"[SHUTDOWN] Failed to shut down Boss: {e}")

def kill_boss_on_port(port=5000):
    try:
        output = subprocess.check_output(["lsof", "-t", f"-i:{port}"])
        for pid in output.decode().splitlines():
            subprocess.run(["kill", "-9", pid])
            print(f"[CLEANUP] Killed boss process {pid}")
    except subprocess.CalledProcessError:
        print("[CLEANUP] No boss process found on port.")

def cleanup_boss():
    shutdown_boss()
    time.sleep(1)
    kill_boss_on_port(5000)


### ------------------ End-to-End Test ------------------ ###

def tester(platform):
    print("\n========================================")
    print(f"Running Boss Test for platform: {platform}")
    print("========================================\n")

    clear_config()
    launch_boss(platform)

    # Step 1: scale to 1 worker (boss may auto-launch 1 on some platforms)
    scale_workers(1)
    wait_for_workers(1)

    # Step 2: upload and verify lambda
    lambda_name = "hi"
    code = ["def f(event):", "\treturn 'hello'"]
    upload_lambda(lambda_name, code)
    verify_lambda_config(lambda_name)

    # Step 3: invoke
    result = invoke_lambda(lambda_name)
    assert result == "hello", f"Unexpected lambda result: {result}"

    # Step 4: scale down and verify unavailability
    shutdown_and_check(lambda_name)

    # Step 5: delete lambda and verify it's gone
    delete_lambda_and_verify(lambda_name)

    print(f"Test passed for platform: {platform}\n")
    cleanup_boss()

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 boss_test.py (aws|azure|gcp|local) [platform2, ...]")
        sys.exit(1)

    for platform in sys.argv[1:]:
        tester(platform)

if __name__ == "__main__":
    main()
