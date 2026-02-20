import json
import heapq
import os
from collections import defaultdict


# TODO: instead of loading deps from a file, use pip-compile on the fly 
def load_deps_json(deps_path=None):
    # if a file is not passed in then search in this dir
    if deps_path is None: 
        deps_path = os.path.join(os.path.dirname(__file__), "deps.json")
    
    if os.path.exists(deps_path):
        with open(deps_path, 'r') as f:
            return json.load(f)
    return {}


class Node:    
    def __init__(self, calls, packages=None):
        self.calls = calls 
        self.packages = packages or set()  # packages pre-loaded at this node
        self.children = []
        self.parent = None
    
    def all_packages(self):
        # gets all the current packages at this node, including those inherited from parents
        if self.parent is None:
            return self.packages.copy()
        return self.parent.all_packages() | self.packages # combine this set with parent's packages
    
    def to_dict(self):
        return {
            "packages": sorted(list(self.packages)),
            "call_count": len(self.calls),
            "children": [child.to_dict() for child in self.children]
        }


class Candidate:    
    def __init__(self, parent, child_pkgV, utility):
        self.parent = parent
        self.child_pkgV = child_pkgV
        self.utility = utility


candidate_queue = [] # priority queue for candidate nodes


def enqueue_top_child_candidate(parent, deps, multi_package=True):
    if not parent.calls:
        return
    
    loaded_pkgs = parent.all_packages() # get all packages currently loaded at this node (including parents)
    loaded_names = {pkg.split("==")[0]: pkg for pkg in loaded_pkgs} # get only names for conflict checking
    
    # get all packages needed by calls in this node
    needed_pkgs = set()
    for pkgs in parent.calls.values(): # equivalent to parent.calls.column_names in forklift paper
        needed_pkgs.update(pkgs)
    
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
        # TODO: see if we can handle multiple versions of the same package
        if pkg_name in loaded_names:
            continue
        
        # keep track of packages that would be loaded by this candidate
        packages_to_load = set([child_pkgV])
        
        # handle multi-package case
        # TODO: double check logic (not sure if this is fully correct)
        if multi_package:
            # check if we have dependency info for this package --> maybe use pip compile instead? 
            if pkg_name in deps and version in deps.get(pkg_name, {}): 
                # get dependencies for this package and version
                dep_sets = deps[pkg_name][version]
                if dep_sets:
                    # dep sets are sorted by call frequency so using the first set picks the one that is called the most often
                    # TODO: might change it to a better approach later instead of choosing first
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
            if p_name in deps and p_version in deps.get(p_name, {}):
                dep_sets = deps[p_name][p_version]
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
        
        # calculate utility: number of calls that would be satisfied by loading these packages
        utility = 0
        for call_pkgs in parent.calls.values():
            if all(pkg in call_pkgs for pkg in packages_to_load):
                utility += 1
        
        if utility == 0:
            continue
        
        # keep track of best candidate
        if utility > best_utility:
            best_utility = utility
            best_candidate = Candidate(parent, packages_to_load, utility)
    
    if best_candidate is not None:
        # push to queue, use negative utility because lower values have higher priority
        heapq.heappush(candidate_queue, (-best_candidate.utility, id(best_candidate), best_candidate)) 


def add_child_node(candidate, deps, multi_package=True):
    parent = candidate.parent
    child_pkgV = candidate.child_pkgV
    
    child_calls = {}
    parent_calls = {}
    
    for call_id, required_pkgs in parent.calls.items():
        if all(pkg in required_pkgs for pkg in child_pkgV):
            remaining = required_pkgs - child_pkgV
            child_calls[call_id] = remaining
        else:
            parent_calls[call_id] = required_pkgs
    
    # create child node and add to parent's children
    child = Node(calls=child_calls, packages=child_pkgV)
    child.parent = parent
    parent.children.append(child)
    
    parent.calls = parent_calls
    
    # enqueue new candidates for parent and child
    enqueue_top_child_candidate(parent, deps, multi_package)
    enqueue_top_child_candidate(child, deps, multi_package)


def build_tree(calls, desired_nodes, deps={}, multi_package=True):
    global candidate_queue
    candidate_queue = []  
        
    # start with empty root node with all calls and no packages (this would be only Python)
    root = Node(calls=calls, packages=set())
    
    # add initial best candidate
    enqueue_top_child_candidate(root, deps, multi_package)
    
    # keep adding nodes to tree until we reach desired size
    while desired_nodes > 0 and candidate_queue:
        # get best candidate from priority queue
        _, _, best_candidate = heapq.heappop(candidate_queue)
        
        # add as child
        add_child_node(best_candidate, deps, multi_package)
        desired_nodes -= 1
    
    return root


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
    calls = {}
    if call_counts:
        for func_name, packages in func_packages.items():
            count = max(1, call_counts.get(func_name, 1))
            for i in range(count):
                calls[f"{func_name}_{i}"] = packages.copy()
    else:
        # if no call counts, just add one call per function
        for func_name, packages in func_packages.items():
            calls[func_name] = packages.copy()
    
    return calls


# TODO: won't be needed if we switch to using pip-compile
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


def generate_tree(workload_data, num_nodes, multi_package=True):
    calls = parse_workload(workload_data)
    deps_json = load_deps_json()
    deps = parse_deps(deps_json)
    
    # build tree
    root = build_tree(calls, num_nodes, deps, multi_package)
    
    return root.to_dict()


def f(event):
    """
    Lambda function entry point for generating Forklift zygote trees.
    
    Expected event format:
    {
        "workload": { ... workload data ... },
        "num_nodes": <int>,
        "multi_package": <bool> (optional, default True)
    }
    
    Returns:
        {
            "status": "success" | "error",
            "result": { ... tree structure ... },
            "error": "..." (if error)
        }
    """
    try:
        if isinstance(event, str):
            event = json.loads(event)
        
        workload_data = event.get("workload")
        if not workload_data:
            return {"status": "error", "error": "Missing 'workload' in event data"}
        
        num_nodes = event.get("num_nodes")
        if not num_nodes:
            return {"status": "error", "error": "Missing 'num_nodes' in event data"}
        
        if not isinstance(num_nodes, int) or num_nodes < 1:
            return {"status": "error", "error": "'num_nodes' must be a positive integer"}
        
        multi_package = event.get("multi_package", True)
        
        result = generate_tree(workload_data, num_nodes, multi_package)
        
        return {
            "status": "success",
            "result": result
        }
        
    except Exception as e:
        import traceback
        return {
            "status": "error",
            "error": str(e),
            "traceback": traceback.format_exc()
        }


# Testing code
if __name__ == "__main__":
    with open("wl.json", "r") as file:
        sample_workload = json.load(file)
    
    result = f({"workload": sample_workload, "num_nodes": 40}) # paper suggests that ~40-80 nodes seems to be optimal
    print(json.dumps(result, indent=2))