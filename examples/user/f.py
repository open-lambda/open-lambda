import subprocess, uuid, errno, os

"""
This test is designed to test disk space exhaustion (ENOSPC) handling.
The worker must be installed on quota enabled disk with testuser
"""
def f(event):
    """
    OpenLambda function to detect ENOSPC and report disk usage.
    """
    df_result = {}
    write_result = {}
   
    try:
        # First, get disk usage
        result = subprocess.run(['df', '/tmp'], capture_output=True, text=True)
        whoami = subprocess.run(['whoami'], capture_output=True, text=True)
       
        if result.returncode == 0:
            lines = result.stdout.strip().split('\n')
            if len(lines) >= 2:
                parts = lines[1].split()
                filesystem = parts[0]
                onekbloks = int(parts[1])
                used_kb = int(parts[2])
                available_kb = int(parts[3])
                use_percent = parts[4]
               
                df_result = {
                    "filesystem": filesystem,
                    "onekbloks": onekbloks,
                    "used_bytes": used_kb,
                    "available_bytes": available_kb,
                    "use_percent": use_percent
                }
        else:
            df_result = {"error": f"df command failed: {result.stderr}"}
   
    except Exception as e:
        df_result = {"error": f"df command error: {str(e)}"}
   
    # Now try to write a file - THIS is where ENOSPC will occur
    try:
        # Write a larger test to trigger ENOSPC more reliably
        test_data = 'X' * 1024 * 1024  # 1MB of data
       
        with open('/tmp/'+str(uuid.uuid4())+'.txt', 'w') as f:
            f.write(test_data)
            f.flush()  # Force write to disk
            os.fsync(f.fileno())  # Ensure data is written to storage
       
        # If we get here, write succeeded
        write_result = {
            "write_status": "success",
            "bytes_written": len(test_data)
        }
       
        # Clean up the test file
        try:
            os.remove('/tmp/openlambda-test.txt')
        except:
            pass
           
    except OSError as e:
        if e.errno == errno.ENOSPC:
            write_result = {
                "write_status": "ENOSPC_ERROR",
                "error_code": e.errno,
                "error_message": "No space left on device",
                "errno_name": "ENOSPC"
            }
        else:
            write_result = {
                "write_status": "OS_ERROR",
                "error_code": e.errno,
                "error_message": str(e)
            }
    except IOError as e:
        if e.errno == errno.ENOSPC:
            write_result = {
                "write_status": "ENOSPC_ERROR_IO",
                "error_code": e.errno,
                "error_message": "No space left on device (IOError)",
                "errno_name": "ENOSPC"
            }
        else:
            write_result = {
                "write_status": "IO_ERROR",
                "error_code": e.errno,
                "error_message": str(e)
            }
    except Exception as e:
        write_result = {
            "write_status": "UNEXPECTED_ERROR",
            "error_message": str(e)
        }
   
    # Return both disk info and write test results
    return {
        "disk_info": df_result,
        "write_test": write_result,
        "user": whoami.stdout.strip()
    }
