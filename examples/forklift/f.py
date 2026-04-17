'''
This Lambda function implements the Forklift algorithm to generate zygote trees for Python workloads.
https://pages.cs.wisc.edu/~yuanzhuo/assets/pdf/forklift.pdf
'''

import heapq
import traceback
from collections import defaultdict, namedtuple
from flask import Flask, request, jsonify

Candidate = namedtuple('Candidate', ['parent', 'child_pkgV', 'utility'])
QueueEntry = namedtuple('QueueEntry', ['neg_utility', 'uid', 'candidate'])

app = Flask(__name__)


class CallMatrix:
    def __init__(self, rows):
        self.rows = rows # {call_id: set(pkg_strings)}

    def __bool__(self):
        return bool(self.rows)
    
    def all_packages(self):
        pkgs = set()
        for row in self.rows.values():
            pkgs.update(row) # add all packages from this call to the set
        return pkgs

    def count_matching(self, packages):
        # counts how many calls require all of the given packages
        return sum(1 for row in self.rows.values() if all(p in row for p in packages))

    def split(self, packages):
        matching = {}
        remaining = {}
        # for each call, check if it requires all of the given packages
        for call_id, required_pkgs in self.rows.items():
            # if it does, add it to the matching matrix with those packages removed from its requirements
            if all(p in required_pkgs for p in packages):
                matching[call_id] = required_pkgs - packages
            # if it doesn't, add it to the remaining matrix
            else:
                remaining[call_id] = required_pkgs
        return CallMatrix(matching), CallMatrix(remaining)
    
    
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
    
    # build call matrix
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
    
    return CallMatrix(rows)
    

def parse_deps(deps_json):
    deps = {}
    
    # the deps file contains a mapping of package to its versions and the dependencies for each version
    for pkg_name, versions in deps_json.items():
        deps[pkg_name] = {}
        for version, dep_strings in versions.items():
            deps[pkg_name][version] = []
            for dep_str in dep_strings.keys():
                dep_packages = set(dep_str.split(",")) if dep_str else set()
                deps[pkg_name][version].append(dep_packages)
    return deps


class ZygoteTree: 
    def __init__(self, calls, deps):
        self.calls = calls
        self.deps = deps
        self.root = None
        self.candidate_queue = []
    
    def _enqueue_top_child_candidate(self, parent):
        if not parent.calls:
            return
        
        loaded_pkgs = parent.all_packages() # get all packages currently loaded at this node (including parents)
        loaded_names = {pkg.split("==")[0]: pkg for pkg in loaded_pkgs} # get only names for conflict checking
        
        # get all packages needed by calls in this node
        needed_pkgs = parent.calls.all_packages()
        
        best_candidate = None
        best_utility = -1
        
        # find best candidate package to load as a child node
        for child_pkgV in needed_pkgs:
            # skip if already loaded
            if child_pkgV in loaded_pkgs:
                continue
            
            pkg_name = child_pkgV.split("==")[0]
            version = child_pkgV.split("==")[1] if "==" in child_pkgV else None
            
            # skip if this package but different version is loaded
            if pkg_name in loaded_names:
                continue
            
            # keep track of packages that would be loaded by this candidate
            packages_to_load = {child_pkgV}
            
            # paper suggests that multipackage trees perform better than single package trees
            if pkg_name in self.deps and version in self.deps.get(pkg_name, {}): 
                # get dependencies for this package and version
                dep_sets = self.deps[pkg_name][version]
                if dep_sets:
                    # dep sets are sorted by call frequency so using the first set picks the one that is called the most often
                    for dep_pkg in dep_sets[0]:
                        if dep_pkg not in loaded_pkgs:
                            dep_name = dep_pkg.split("==")[0]
                            # check for conflicts with loaded packages
                            if dep_name not in loaded_names:
                                packages_to_load.add(dep_pkg)
            
            # make sure that packages are valid
            # according to the Forklift paper:
            # a package P is valid for a node N if the ancestor nodes of N are responsible for loading all of P's dependencies.
            valid = True
            for pkg in packages_to_load:
                p_name = pkg.split("==")[0]
                p_version = pkg.split("==")[1] if "==" in pkg else None
                if p_name in self.deps and p_version in self.deps.get(p_name, {}):
                    dep_sets = self.deps[p_name][p_version]
                    if dep_sets:
                        for dep in dep_sets[0]:
                            # make sure that this dependency is either already loaded by an ancestor or would be loaded by this candidate
                            if dep not in loaded_pkgs and dep not in packages_to_load:
                                valid = False
                                break
                if not valid:
                    break
            
            if not valid:
                continue
            
            # calculate utility: len(packages_to_load) × matching_calls
            # assuming equal weigths for all packages
            matching_calls = parent.calls.count_matching(packages_to_load)
            utility = len(packages_to_load) * matching_calls
            
            if utility == 0:
                continue
            
            # keep track of best candidate
            if utility > best_utility:
                best_utility = utility
                best_candidate = Candidate(parent, packages_to_load, utility)
        
        if best_candidate is not None:
            # push to queue
            entry = QueueEntry(
                neg_utility=-best_candidate.utility, # use negative utility because we're using a min heap
                uid=id(best_candidate), # using id to sort candidates that have the same utility
                candidate=best_candidate,
            )
            heapq.heappush(self.candidate_queue, entry)

    def _add_child_node(self, candidate):
        parent = candidate.parent
        child_pkgV = candidate.child_pkgV
        
        child_calls, parent_calls = parent.calls.split(child_pkgV)
        
        # create child node and add to parent's children
        child = ZygoteNode(calls=child_calls, packages=child_pkgV)
        child.parent = parent
        parent.children.append(child)
        
        parent.calls = parent_calls
        
        # enqueue new candidates for parent and child
        self._enqueue_top_child_candidate(parent)
        self._enqueue_top_child_candidate(child)
    
    def build_tree(self, desired_nodes):
        self.candidate_queue = []  
            
        # start with empty root node with all calls and no packages
        self.root = ZygoteNode(self.calls, set())
        
        # add initial best candidate
        self._enqueue_top_child_candidate(self.root)
        
        # keep adding nodes to tree until we reach desired size
        while desired_nodes > 0 and self.candidate_queue:
            # get best candidate from priority queue
            entry = heapq.heappop(self.candidate_queue)
            
            # add as child
            self._add_child_node(entry.candidate)
            desired_nodes -= 1       

    def to_dict(self):
        return self.root.to_dict()


class ZygoteNode:    
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
        if event is None:
            return jsonify({"error": "Request body must be valid JSON"}), 400

        workload_data = event.get("workload")
        deps_data = event.get("deps")
        num_nodes = event.get("num_nodes")

        if workload_data is None or deps_data is None or num_nodes is None:
            return jsonify({"error": "Missing required fields: workload, deps, num_nodes"}), 400

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