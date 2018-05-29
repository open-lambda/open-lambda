import json
import unittest
from cluster_test_utils import start_test_cluster, run_cluster_test_with_conf 


class DockerTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        start_test_cluster()

    def test_with_no_cache(self):
        conf = json.dumps({
            'sandbox': 'docker', 
            'handler_cache_size': 0, 
            'import_cache_size': 0, 
            'cg_pool_size': 0
        })        
        run_cluster_test_with_conf(conf)

    def test_with_handler_cache(self):
        conf = json.dumps({
            'sandbox': 'docker', 
            'handler_cache_size': 10000000, 
            'import_cache_size': 0, 
            'cg_pool_size': 0
        })        
        run_cluster_test_with_conf(conf)

if __name__ == '__main__':
    unittest.main()

