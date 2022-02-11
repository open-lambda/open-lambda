import sys
import platform


def check_versions():
    version = sys.version
    return version

# return Ubuntu and Python versions to be printed to console
def f(event):
    #ubuntu_version =  platform.linux_distribution()[1]
    python_version = sys.version
    return python_version

if __name__ == "__main__":
    cv = check_versions()
