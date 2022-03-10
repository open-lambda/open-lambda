''' OpenLambda's Python API '''

import requests
import json

class OpenLambda:
    ''' Represents a client connection to OpenLambda '''

    def __init__(self, address = "localhost:5000"):
        self._address = address

    def post(self, path, data = None):
        ''' Issues a post request to the OL worker '''
        return requests.post(f'http://{self._address}/{path}', json.dumps(data))

    def run(self, fn_name, args, json=True):
        ''' Execute a serverless function '''

        req = self.post(f"run/{fn_name}", args)
        if req.status_code != 200:
            raise Exception(f"STATUS {req.stats_code}: {req.text}")

        if json:
            return req.json()
        else:
            return req.text

    def run_on(self, object_id, fn_name, args):
        ''' Execute a serverless function on a LambdaObject '''

        req = self.post(f"run/{fn_name}?object_id={object_id}", args)
        if req.status_code != 200:
            raise Exception(f"STATUS {req.stats_code}: {req.text}")

        return req.json()

    def get_status(self):
        ''' Returns the status of the OpenLambda server '''
        req = requests.get(f"http://{self._address}/status")
        if req.status_code != 200:
            raise Exception(f"STATUS {req.stats_code}: {req.text}")
