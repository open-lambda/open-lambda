package loadbalancer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"
)

type Node struct {
	Direct          []string `json:"direct"`
	Packages        []string `json:"packages"`
	Children        []*Node  `json:"children"`
	SplitGeneration int      `json:"split_generation"`
	Count           int      `json:"count"`
	SubtreeCount    int      `json:"subtree_count"`
}

var root *Node
var shardLists [][][]*Node

// // index is the cluster id, value is the cluster/shard's root node
// var shardingList1 = []int{3}
// var shardingList2 = []int{3}
// var shardingList3 = []int{9, 2, 38, 37, 51, 147, 79, 143, 182, 82}
// var shardingList4 = []int{4, 25, 81, 46, 57, 100, 15, 85, 169, 197, 103}
// var shardingList5 = []int{1, 124, 36, 17, 108, 144, 123, 28, 109, 179, 66, 106}

// // These are all the split_generations, or ids
// var ShardingLists = [][]int{
// 	{3},
// 	{3},
// 	{9, 2, 38, 37, 51, 147, 79, 143, 182, 82},
// 	{4, 25, 81, 46, 57, 100, 15, 85, 169, 197, 103},
// 	{1, 124, 36, 17, 108, 144, 123, 28, 109, 179, 66, 106},
// }

// BySubtreeCount implements sort.Interface for []*Node based on the SubtreeCount field.
type BySubtreeCount []*Node

func (a BySubtreeCount) Len() int           { return len(a) }
func (a BySubtreeCount) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a BySubtreeCount) Less(i, j int) bool { return a[i].SubtreeCount > a[j].SubtreeCount }

func splitNodes(nodes []*Node, n int) ([][]*Node, []int) {
	// Sort the nodes by subtree_count in descending order
	sort.Sort(BySubtreeCount(nodes))

	// Initialize n sets
	sets := make([][]*Node, n)
	setSums := make([]int, n) // To keep track of the sum of subtree_count for each set

	// Distribute nodes into sets
	for _, node := range nodes {
		// Find the set with the smallest sum
		minSetIdx := 0
		for i := 1; i < n; i++ {
			if setSums[i] < setSums[minSetIdx] {
				minSetIdx = i
			}
		}
		// Add the current node to the selected set
		sets[minSetIdx] = append(sets[minSetIdx], node)
		// Update the sum of the selected set
		setSums[minSetIdx] += node.SubtreeCount
	}

	return sets, setSums
}

func splitTree(n int, m int) [][][]*Node {
	var nodes []*Node
	nodes = append(nodes, root.Children...)

	keepSplit := true
	var sets [][]*Node
	var setSums []int
	depth := 0
	var setsSumsDict [][][]*Node

	for keepSplit {
		depth++
		if depth > m {
			break
		}
		keepSplit = false
		sets, setSums = splitNodes(nodes, n)
		minSum := min(setSums)
		setsSumsDict = [][][]*Node{}

		for i, set := range sets {
			setSum := setSums[i]
			if float64(setSum) > 1.1*float64(minSum) {
				keepSplit = true
				for _, node := range set {
					nodes = removeNode(nodes, node)
					nodes = append(nodes, node.Children...)
				}
			} else {
				setsSumsDict = append(setsSumsDict, [][]*Node{set, {&Node{SubtreeCount: setSum}}})
			}
		}
	}

	return setsSumsDict
}

func min(sums []int) int {
	minValue := sums[0]
	for _, v := range sums {
		if v < minValue {
			minValue = v
		}
	}
	return minValue
}

func removeNode(nodes []*Node, target *Node) []*Node {
	var result []*Node
	for _, node := range nodes {
		if node != target {
			result = append(result, node)
		}
	}
	return result
}

func UpdateShard(n, m int) {
	// Call splitTree to get the sets
	if n == 0 {
		return
	}
	sets := splitTree(n, m)

	// Add these sets to the global shardLists
	shardLists = make([][][]*Node, 0)
	shardLists = append(shardLists, sets...)

	for i, setSum := range shardLists {
		sum := setSum[1][0].SubtreeCount
		set := setSum[0]

		subtreeCounts := make([]string, len(set))
		splitGenerations := make([]string, len(set))
		for j, node := range set {
			subtreeCounts[j] = fmt.Sprintf("%d", node.SubtreeCount)
			splitGenerations[j] = fmt.Sprintf("%d", node.SplitGeneration)
		}

		fmt.Printf("Set %d has a sum of %d and contains nodes with subtree_counts: [%s] with ids: [%s]\n", i+1, sum, strings.Join(subtreeCounts, ", "), strings.Join(splitGenerations, ", "))
	}
	fmt.Println()
}

func updateSubtreeCount(node *Node) int {
	// Base case: if the node has no children, its subtree_count is just its own count
	if len(node.Children) == 0 {
		node.SubtreeCount = node.Count
		return node.Count
	}

	// Start with the current node's count
	totalCount := node.Count

	// Recursively update the count for all children
	for _, child := range node.Children {
		totalCount += updateSubtreeCount(child)
	}

	// After the total count for all children is calculated, update the current node's subtree_count
	node.SubtreeCount = totalCount

	return totalCount
}

func GetRoot() error {
	// Read the JSON file
	// TODO: not to hardcode
	fileContent, err := ioutil.ReadFile(tree_path)
	if err != nil {
		return err
	}

	// Unmarshal the JSON content into the Node struct
	rootNode := Node{}
	err = json.Unmarshal(fileContent, &rootNode)
	if err != nil {
		return err
	}
	root = &rootNode

	// update the subtree_count
	updateSubtreeCount(root)

	return nil
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
	// for _, pkg := range pkgs {
	// 	fmt.Println(pkg)
	// }
	_, path := root.Lookup(pkgs)
	// for _, node := range path {
	// 	fmt.Println(node.SplitGeneration)
	// }
	for _, node := range path {
		for i, setSum := range shardLists {
			set := setSum[0]
			for _, shardNode := range set {
				if shardNode.SplitGeneration == node.SplitGeneration {
					return i, nil
				}
			}
		}
	}
	return -1, nil
}
