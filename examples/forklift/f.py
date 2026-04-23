'''
This Lambda function implements the Forklift algorithm to generate zygote trees for Python workloads.
https://pages.cs.wisc.edu/~yuanzhuo/assets/pdf/forklift.pdf
'''

import heapq
import traceback
from collections import defaultdict, namedtuple
import pandas as pd
from flask import Flask, request, jsonify

Candidate = namedtuple('Candidate', ['parent', 'child_pkgV', 'utility'])
QueueEntry = namedtuple('QueueEntry', ['neg_utility', 'uid', 'candidate'])

app = Flask(__name__)


def parse_workload(workload):
    func_packages = {}

    for func in workload.get("funcs", []):
        func_name = func.get("name", "")
        if not func_name:
            continue
        
        packages = set()
        meta = func.get("meta", {})
        req_txt = meta.get("requirements_txt", "")
        
        for line in req_txt.split("\n"):
            line = line.strip()
            if "==" in line and not line.startswith("#"):
                packages.add(line)
        
        if packages:
            func_packages[func_name] = packages
    
    # count call frequencies
    call_counts = defaultdict(int)
    for call in workload.get("calls", []):
        name = call.get("name", "")
        if name:
            call_counts[name] += 1
    
    # rows=calls, columns=package==version, values=0/1
    rows = {}
    if call_counts:
        for func_name, packages in func_packages.items():
            count = max(1, call_counts.get(func_name, 1))
            for i in range(count):
                rows[f"{func_name}_{i}"] = packages.copy()
    else:
        # if no call counts, just add one call per function
        for func_name, packages in func_packages.items():
            rows[func_name] = packages.copy()
    
    if not rows:
        return pd.DataFrame(dtype=int)
    
    # sorted list of all unique packages across all calls, to ensure consistent column ordering
    all_pkgs = sorted(set().union(*rows.values()))
    # create DataFrame with 1s for packages used by each call, 0s otherwise (sparse representation)
    records = [{pkg: 1 for pkg in pkgs} for pkgs in rows.values()]
    # reindex to ensure all columns are present and in the same order, filling missing values with 0 (dense matrix)
    df = pd.DataFrame(records, index=list(rows.keys()))
    df = df.reindex(columns=all_pkgs, fill_value=0).fillna(0).astype(int)
    return df
    

def parse_deps(deps_json):
    deps = {}
    for pkg_name, versions in deps_json.items():
        for version, dep_strings in versions.items():
            pkgV = f"{pkg_name}=={version}"
            deps[pkgV] = []
            for dep_str in dep_strings.keys():
                dep_packages = set(dep_str.split(",")) if dep_str else set()
                deps[pkgV].append(dep_packages)
    return deps


class ZygoteTree: 
    def __init__(self, calls, deps):
        self.calls = calls
        self.deps = deps
        self.root = None
        self.candidateQ = []
    
    def check_candidate_validity(self, parent, child_pkgV):
        loaded_pkgs = parent.all_packages()
        loaded_names = {p.split("==")[0] for p in loaded_pkgs}
        # skip if already loaded or version conflict with ancestor
        if child_pkgV in loaded_pkgs or child_pkgV.split("==")[0] in loaded_names:
            return False
        # all dependencies of child_pkgV must be satisfied by ancestors
        dep_sets = self.deps.get(child_pkgV, [])
        if dep_sets:
            for dep in dep_sets[0]:
                if dep != child_pkgV and dep not in loaded_pkgs:
                    return False
        return True

    def enqueue_top_child_candidate(self, parent):
        best_candidate = None
        
        for child_pkgV in parent.calls.columns:
            if self.check_candidate_validity(parent, child_pkgV):
                # utility = sum of the package column (usage frequency)
                utility = int(parent.calls[child_pkgV].sum())
                # keep track of the best candidate with the highest utility
                if utility > 0 and (best_candidate is None or utility > best_candidate.utility):
                    best_candidate = Candidate(parent, child_pkgV, utility)
        
        # push the best candidate for this parent into the priority queue
        if best_candidate is not None:
            heapq.heappush(self.candidateQ, QueueEntry(-best_candidate.utility, id(best_candidate), best_candidate))

    def add_child_node(self, candidate):
        parent = candidate.parent
        child_pkgV = candidate.child_pkgV
        
        # rows that import child_pkgV move to the child, remaining rows stay with the parent
        child_calls = parent.calls[parent.calls[child_pkgV] != 0].copy()
        
        child = Node(calls=child_calls, packages={child_pkgV})
        child.parent = parent
        parent.children.append(child)
        
        parent.calls = parent.calls.drop(child_calls.index).copy()
        
        self.enqueue_top_child_candidate(parent)
        self.enqueue_top_child_candidate(child)
    
    def build_tree(self, desired_nodes):
        self.candidateQ = [] # priority queue of candidates with highest utility first
            
        # start from a root with all calls and no preloaded packages.
        self.root = Node(self.calls, set())
        
        # initialize the candidate queue with the root's best child candidate
        self.enqueue_top_child_candidate(self.root)
        
        while desired_nodes > 0 and self.candidateQ:
            best_candidate = heapq.heappop(self.candidateQ).candidate
            self.add_child_node(best_candidate)
            desired_nodes -= 1

    def to_dict(self):
        return self.root.to_dict()


class Node:    
    def __init__(self, calls, packages=None, parent=None):
        self.calls = calls 
        self.packages = packages or set()  # packages pre-loaded at this node
        self.children = []
        self.parent = parent
    
    def all_packages(self):
        # gets all the current packages at this node, including those inherited from parents
        if self.parent is None:
            return self.packages.copy()
        return self.parent.all_packages() | self.packages # combine this set with parent's packages
    
    def to_dict(self):
        return {
            "packages": sorted(list(self.packages)),
            "children": [child.to_dict() for child in self.children]
        }


@app.route("/", methods=["POST"])
def f():
    '''
    Expected input format:

    {
        "workload": {
            "funcs": [
                {
                    "name": "function_name",
                    "meta": {
                        "requirements_txt": "package1==version\npackage2==version\n..."
                    }
                },
                ...
            ],
            "calls": [
                {"name": "function_name"},
                ...
            ]
        },
        "deps": {
            <package_name>: {
                <version>: {
                    <comma_separated_deps>: <call_frequency>,
                    ...
                },
                ...
            },
            ...
        },
        "num_nodes": <int>
    }
    '''

    try:
        event = request.get_json()

        workload_data = event.get("workload")
        deps_data = event.get("deps")
        num_nodes = event.get("num_nodes")

        # parse inputs
        calls = parse_workload(workload_data)
        deps = parse_deps(deps_data)

        # build tree
        tree = ZygoteTree(calls, deps)
        tree.build_tree(num_nodes)

        result = tree.to_dict()

        return jsonify(result), 200

    except Exception as e:
        return jsonify({
            "error": str(e),
            "traceback": traceback.format_exc()
        }), 500