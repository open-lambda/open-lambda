''' OpenLambda's Python API '''

import json as pyjson
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

        req = self._post(f"run/{fn_name}", args)
        self._check_status_code(req, "run")

        if json:
            return req.json()

        return req.text

    def create(self, args):
        ''' Create a new sandbox '''

        req = self._post("create", args)
        self._check_status_code(req, "create")
        return req.text.strip()

    def destroy(self, sandbox_id):
        ''' Destroy a new sandbox '''

        req = self._post(f"destroy/{sandbox_id}", {})
        self._check_status_code(req, "destroy")

    def pause(self, sandbox_id):
        ''' Pause a new sandbox '''

        req = self._post(f"pause/{sandbox_id}", None)
        self._check_status_code(req, "pause")

    def get_statistics(self):
        ''' Returns stats of the OpenLambda server '''

        req = self._session.get(f"http://{self._address}/stats")
        self._check_status_code(req, "get_statistics")
        return req.json()

    @staticmethod
    def _check_status_code(req, name):
        if req.status_code != 200:
            raise Exception(f'"{name}" failed with status code {req.status_code}: {req.text}')

    def check_status(self):
        ''' Checks the status of the OpenLambda server '''
        req = self._session.get(f"http://{self._address}/status")
        self._check_status_code(req, "check_status")
