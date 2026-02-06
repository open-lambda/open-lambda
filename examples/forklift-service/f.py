import json
import sys
import os
from pathlib import Path
import tempfile

# try to get path from env vars
FORKLIFT_PATH = os.environ.get('FORKLIFT_PATH')
# otherwise, assume it is a sibling dir to open lambda
if FORKLIFT_PATH is None:
    FORKLIFT_PATH = str(Path(__file__).parent.parent.parent.parent / "forklift" / "tree_constructor")

# add to path to be able to import from forklift
if FORKLIFT_PATH not in sys.path:
    sys.path.insert(0, FORKLIFT_PATH)

# import Forklift modules
from bench import Tree, SplitOpts
from workload import Workload
from version import Package

# TODO might need to be updated to reflect newer packages? 
PACKAGES = os.path.join(FORKLIFT_PATH, "packages.json") # path to json file contianing known package information


def filter_workload_by_known_packages(workload_data, known_packages):
    filtered_funcs = []
    removed_funcs = []
    valid_func_names = set()
    
    for func in workload_data.get("funcs", []):
        # get packages used by this function from meta
        func_packages = set()
        meta = func.get("meta", {})
        
        # extract package names from requirements_txt since it also has indirect dependencies
        req_txt = meta.get("requirements_txt")
        for line in req_txt.split("\n"):
            line = line.strip()
            if "==" in line and not line.startswith("#"):
                pkg_name = line.split("==")[0].strip()
                if pkg_name:
                    func_packages.add(pkg_name)
        
        # check if packages are in the known list
        unknown_packages = func_packages - known_packages
        if unknown_packages:
            # removed the unknown packages
            removed_funcs.append({
                "name": func.get("name"),
                "unknown_packages": list(unknown_packages)
            })
        else:
            filtered_funcs.append(func)
            valid_func_names.add(func.get("name"))
    
    # filter calls to only include valid functions
    filtered_calls = [
        call for call in workload_data.get("calls", [])
        if call.get("name") in valid_func_names
    ]
    
    # filter pkg_with_version to only include known packages
    filtered_pkg_with_version = {
        pkg: versions for pkg, versions in workload_data.get("pkg_with_version", {}).items()
        if pkg in known_packages
    }
    
    filtered_workload = {
        "funcs": filtered_funcs,
        "calls": filtered_calls,
        "pkg_with_version": filtered_pkg_with_version
    }
    
    return filtered_workload, removed_funcs


def generate_tree(workload_data, num_nodes):
    # load package json file
    if os.path.exists(PACKAGES):
        Package.from_json(PACKAGES)
    
    # get known packages as a set and filter out unknown ones from the workload
    known_packages = set(Package.packages_factory.keys())
    filtered_workload, removed_funcs = filter_workload_by_known_packages(workload_data, known_packages)

    if removed_funcs:
        print(f"Filtered out {len(removed_funcs)} functions with unknown packages")
        # print the names of (unique) packages that were filtered out
        unique_unknown_packages = set()
        for func in removed_funcs:
            unique_unknown_packages.update(func["unknown_packages"])
        print(f"Unknown packages: {', '.join(unique_unknown_packages)}")
    
    # workload expects a file path so we need to temporarily store the data
    # TODO maybe we should change forklift to accept a dict directly?
    with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
        json.dump(filtered_workload, f)
        temp_workload_path = f.name
    
    try:
        # load workload from temp file
        wl = Workload(temp_workload_path)
        
        # create tree
        # TODO maybe add option to pass in dict as a parameter to the lambda function to change these?
        opts = SplitOpts(
            costs=Package.cost_dict(), # maps package names to costs (packages that take longer are prioritized)
            prereq_first=True, # prioritize functions with prerequisites first
            entropy_penalty=0, # penalty for creating unbalanced splits
            avg_dist_weights=False, # use average distribution weights
            biased_dist_weights=False # use biased distribution weights
        )
        
        # generate call and dependency matrices
        call_mat = wl.call_matrix()
        dep_mat = wl.dep_matrix(Package.packages_factory)
        
        # create tree
        tree = Tree(call_mat, dep_mat, wl, opts)
        
        # do splits to create tree nodes
        tree.do_splits(num_nodes - 1) 
        
        return tree.root.to_dict()
        
    finally:
        if os.path.exists(temp_workload_path):
            os.unlink(temp_workload_path)


def f(event):
    try:
        if isinstance(event, str):
            event = json.loads(event)
        # otherwise, assume it is already a dict

        # get workload from input
        workload_data = event.get("workload")
        if not workload_data:
            return {"error": "Missing 'workload' in event data"}
        
        # get number of nodes from input
        num_nodes = event.get("num_nodes")
        if not num_nodes:
            return {"error": "Missing 'num_nodes' in event data"}
        
        # generate tree
        result = generate_tree(workload_data, num_nodes=num_nodes)
        
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


# testing code
if __name__ == "__main__":
    with open(os.path.join(FORKLIFT_PATH, "workloads.json")) as file:
        workload = json.load(file)

    # with open("sample_workload.json") as file:
    #     workload = json.load(file)
    
    tree = f({"workload": workload, "num_nodes": 5})
    print(json.dumps(tree, indent=2))