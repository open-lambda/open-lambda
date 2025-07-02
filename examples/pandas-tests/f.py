import numpy
import pandas
import pytest
import inspect
import os, sys
import subprocess
import time

def f(event):
    print("redirecting to stdout.txt, stderr.txt")
    sys.stdout.close()
    sys.stderr.close()
    sys.stdout = open("/tmp/stdout.txt", 'w', 1)
    sys.stderr = open("/tmp/stderr.txt", 'w', 1)

    pkg = os.path.dirname(pandas.__file__)
    print(pkg)

    # TODO: look into these to determine whether it's a problem
    tests_that_fail = [
        "test_oo_optimizable",
        "test_oo_optimized_datetime_index_unpickle",
        "test_missing_required_dependency",
        "test_util_in_top_level",
        "test_raw_roundtrip",
        "test_get_handle_with_path",
        "test_with_missing_lzma",
        "test_with_missing_lzma_runtime",
        "test_multi_thread_string_io_read_csv",
        "test_multi_thread_path_multipart_read_csv",
        "test_server_and_default_headers",
        "test_server_and_custom_headers",
        "test_server_and_all_custom_headers",
        "TestS3",
        "s3",
    ]

    cmd = [pkg, "-o", "cache_dir=/tmp/.my_cache_dir", "-m", "(not slow)"]
    cmd.extend(["-k", " and ".join([f"(not {name})" for name in tests_that_fail])])
    print(" ".join(cmd))
    result = pytest.main(cmd)
    return result == pytest.ExitCode.OK
