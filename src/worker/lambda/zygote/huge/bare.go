package huge

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/open-lambda/open-lambda/ol/common"
)

// bare.go just manipulates the tree structure with some search
// algorithms.  Does not include any sandboxes and does not require
// locking (it is immutable).

type Node struct {
	// assigned via pre-order traversal, starting at 0
	ID int
	
	// parse from JSON
	Packages []string `json:"packages"`
	Children []*Node  `json:"children"`
}

// LoadTreeFromConfig returns a list of Node pointers upon success.
// Each Node has an ID corresponding to its index.  The Node at index
// 0 is the root.
func LoadTreeFromConfig() ([]*Node, error) {
	var root *Node = &Node{};
	var err error;

	switch treeConf := common.Conf.Import_cache_tree.(type) {
	case string:
		if treeConf != "" {
			var b []byte
			if strings.HasPrefix(treeConf, "{") && strings.HasSuffix(treeConf, "}") {
				b = []byte(treeConf)
			} else {
				b, err = ioutil.ReadFile(treeConf)
				if err != nil {
					return nil, fmt.Errorf("could not open import tree file (%v): %v\n", treeConf, err.Error())
				}
			}

			if err := json.Unmarshal(b, root); err != nil {
				return nil, fmt.Errorf("could parse import tree file (%v): %v\n", treeConf, err.Error())
			}
		}
	case map[string]any:
		b, err := json.Marshal(treeConf)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(b, root); err != nil {
			return nil, err
		}
	}

	// assign every node an ID, 0-N
	nodes := []*Node{}
	recursiveNodeInit(root, &nodes)

	return nodes, nil
}

func recursiveNodeInit(node *Node, nodes *[]*Node) {
	node.ID = len(*nodes)
	*nodes = append(*nodes, node)

	for _, child := range node.Children {
		recursiveNodeInit(child, nodes)
	}
}

// isSubset returns true iff every item in A is also in B
func isSubset(A []string, B []string) bool {
	for _, a := range A {
		found := false
		for _, b := range B {
			if a == b {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// findDiff returns items in A that are not in B
func findDiff(A []string, B []string) []string {
	diff := []string{}
	for _, a := range A {
		found := false
		for _, b := range B {
			if a == b {
				found = true
				break
			}
		}
		if !found {
			diff = append(diff, a)
		}
	}
	return diff
}

// FindEligibleZygotes finds IDs of Zygotes that could be used to
// create a Sandbox with the desired packages.  A Zygote is only
// eligible if it (and it's ancestors) together have a subset of the
// desired package set (don't want to expose functions to packages
// they don't want).
//
// eligible will be populated with IDs along a path from root node to
// most specific Zygote.  eligible[i] is the parent of eligible[i+1].
func (node *Node) FindEligibleZygotes(packages []string, eligible *[]int) bool {
	// if this node imports a package that's not wanted by the
	// lambda, neither this Zygote nor its children will work
	if !isSubset(node.Packages, packages) {
		// this Zygote is not eligible because the
		// node has a package not desired by the
		// sandbox
		return false
	}

	// this node is eligible
	*eligible = append(*eligible, node.ID)

	// check our descendents; is one of them a Zygote that works?
	// we prefer a child Zygote over the one for this node,
	// because they have more packages pre-imported
	remainingPackages := findDiff(packages, node.Packages)
	for _, child := range node.Children {
		if child.FindEligibleZygotes(remainingPackages, eligible) {
			// we prefer the first child in the list, and
			// want a single list of zygotes (from root to
			// lower node), so if we found one that is
			// eligible, don't continue more'
			break
		}
	}

	// this node is eligible
	return true
}
