package loadbalancer

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"time"
)

type Node struct {
	Direct          []string `json:"direct"`
	Packages        []string `json:"packages"`
	Children        []*Node  `json:"children"`
	SplitGeneration int      `json:"split_generation"`
	Count           int      `json:"count"`
}

// index is the cluster id, value is the cluster/shard's root node
var shardingList1 = []int{3}
var shardingList2 = []int{3}
var shardingList3 = []int{9, 2, 38, 37, 51, 147, 79, 143, 182, 82}
var shardingList4 = []int{4, 25, 81, 46, 57, 100, 15, 85, 169, 197, 103}
var shardingList5 = []int{1, 124, 36, 17, 108, 144, 123, 28, 109, 179, 66, 106}

// These are all the split_generations, or ids
var ShardingLists = [][]int{
	{3},
	{3},
	{9, 2, 38, 37, 51, 147, 79, 143, 182, 82},
	{4, 25, 81, 46, 57, 100, 15, 85, 169, 197, 103},
	{1, 124, 36, 17, 108, 144, 123, 28, 109, 179, 66, 106},
}

func getRoot() (*Node, error) {
	// Read the JSON file
	// TODO: not to hardcode
	fileContent, err := ioutil.ReadFile("/home/azureuser/paper-tree-cache/analysis/16/trials/0/tree-v4.node-200.json")
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON content into the Node struct
	var rootNode Node
	err = json.Unmarshal(fileContent, &rootNode)
	if err != nil {
		return nil, err
	}

	return &rootNode, nil
}

func (n *Node) Lookup(required_pkgs []string) (*Node, []*Node) {
	for _, pkg := range n.Packages {
		found := false
		for _, req := range required_pkgs {
			if pkg == req {
				found = true
				break
			}
		}
		if !found {
			return nil, nil
		}
	}

	for _, child := range n.Children {
		bestNode, path := child.Lookup(required_pkgs)
		if bestNode != nil {
			return bestNode, append([]*Node{n}, path...)
		}
	}

	return n, []*Node{n}
}

// if return -1, means no group found, need to randomly choose one
func ShardingGetGroup(pkgs []string) (int, error) {
	root, err := getRoot()
	if err != nil {
		return -1, err
	}
	_, path := root.Lookup(pkgs)
	for _, node := range path {
		for i, shard := range ShardingLists {
			for _, id := range shard {
				if id == node.SplitGeneration {
					if i == 0 { // since two groups has the same, randomly distribute
						rand.Seed(time.Now().UnixNano())
						randomInt := rand.Intn(2)
						return id + randomInt, nil
					}
					return id, nil
				}
			}
		}
	}
	return -1, nil
}
