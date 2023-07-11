''' OpenLambda's Python API '''

import json as pyjson
import requests
from requests import Session

class OpenLambda:
    ''' Represents a client connection to OpenLambda '''

    def __init__(self, address="localhost:5000"):
        self._address = address
        self._session = Session()

    def _post(self, path, data=None):
        ''' Issues a _post request to the OL worker '''
        return self._session.post(f'http://{self._address}/{path}', pyjson.dumps(data))

    def run(self, fn_name, args, json=True):
        ''' Execute a serverless function '''

        resp = self._post(f"run/{fn_name}", args)
        self._check_status_code(resp, "run")

        if json:
            return resp.json()

        return resp.text

    def create(self, args):
        ''' Create a new sandbox '''

        resp = self._post("create", args)
        self._check_status_code(resp, "create")
        return resp.text.strip()

    def destroy(self, sandbox_id):
        ''' Destroy a new sandbox '''

        resp = self._post(f"destroy/{sandbox_id}", {})
        self._check_status_code(resp, "destroy")

    def pause(self, sandbox_id):
        ''' Pause a new sandbox '''

        resp = self._post(f"pause/{sandbox_id}", None)
        self._check_status_code(resp, "pause")

    def get_statistics(self):
        ''' Returns stats of the OpenLambda server '''

        resp = self._session.get(f"http://{self._address}/stats")
        self._check_status_code(resp, "get_statistics")
        return resp.json()

    @staticmethod
    def _check_status_code(resp, name):
        if resp.status_code != 200:
            msg = f'"{name}" failed with status code {resp.status_code}: {resp.text}'
            raise requests.HTTPError(msg)

    def check_status(self):
        ''' Checks the status of the OpenLambda server '''
        resp = self._session.get(f"http://{self._address}/status")
        self._check_status_code(resp, "check_status")
