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
	ParentInt       int      `json:"parent"`

	SubtreeCount int
	Parent       *Node
	Shards       []int
}

var root *Node

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
		fmt.Println(len(sets))
		minSum := min(setSums)
		setsSumsDict = [][][]*Node{}

		for i, set := range sets {
			setSum := setSums[i]
			if depth < m && float64(setSum) > 1.2*float64(minSum) {
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

// contains checks if a slice contains a specific element.
func contains(slice []int, element int) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}

func (n *Node) appendToParents(i int) {
	for node := n; node != nil; node = node.Parent {
		if !contains(node.Shards, i) {
			node.Shards = append(node.Shards, i)
		}
	}
}

func (n *Node) appendToSubtree(i int) {
	if !contains(n.Shards, i) {
		n.Shards = append(n.Shards, i)
	}
	for _, child := range n.Children {
		child.appendToSubtree(i)
	}
}

func (n *Node) clearShards() {
	n.Shards = make([]int, 0)
	for _, child := range n.Children {
		child.clearShards()
	}
}

func UpdateShard(n, m int) {
	// Call splitTree to get the sets
	if n == 0 {
		return
	}
	root.clearShards()
	sets := splitTree(n, m)

	// Add these sets to the global shardLists

	for i, setSum := range sets {
		sum := setSum[1][0].SubtreeCount
		set := setSum[0]

		subtreeCounts := make([]string, len(set))
		splitGenerations := make([]string, len(set))
		for j, node := range set {
			subtreeCounts[j] = fmt.Sprintf("%d", node.SubtreeCount)
			splitGenerations[j] = fmt.Sprintf("%d", node.SplitGeneration)
			// for node's parent: append i to shards field
			// for node's children: append i to shards field
			node.appendToParents(i)
			node.appendToSubtree(i)
		}

		fmt.Printf("Set %d has a sum of %d and contains nodes with subtree_counts: [%s] with ids: [%s]\n", i+1, sum, strings.Join(subtreeCounts, ", "), strings.Join(splitGenerations, ", "))
	}
	fmt.Println()
	// bfs(root)
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

// setParents traverses the tree and sets each node's Parent field.
func setParents(root *Node, generationToNode map[int]*Node) {
	if root == nil {
		return
	}

	// Map the current node's SplitGeneration to the node itself.
	generationToNode[root.SplitGeneration] = root

	// Set the Parent for each child and recurse.
	for _, child := range root.Children {
		child.Parent = generationToNode[child.ParentInt]
		setParents(child, generationToNode)
	}
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

	generationToNode := make(map[int]*Node)

	// Set the parent nodes using the map and recursive traversal.
	setParents(root, generationToNode)
	// fmt.Println(root.Children[0].Children[0].Parent.SplitGeneration)

	// update the subtree_count
	updateSubtreeCount(root)

	return nil
}

func (n *Node) Lookup(required_pkgs []string) *Node {
	for _, pkg := range n.Packages {
		found := false
		for _, req := range required_pkgs {
			if pkg == req {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	for _, child := range n.Children {
		bestNode := child.Lookup(required_pkgs)
		if bestNode != nil {
			return bestNode
		}
	}

	return n
}

func ShardingGetGroup(pkgs []string) ([]int, error) {
	node := root.Lookup(pkgs)
	// fmt.Println("Debug3: ", node.SplitGeneration, node.Shards)
	return node.Shards, nil
}

// func bfs(root *Node) {
// 	queue := []*Node{root} // Initialize the queue with the root node

// 	for len(queue) > 0 {
// 		current := queue[0]                                  // Get the first node in the queue
// 		queue = queue[1:]                                    // Dequeue the current node
// 		fmt.Println(current.SplitGeneration, current.Shards) // Print the Shards of the current node

// 		// Enqueue the children of the current node
// 		for _, child := range current.Children {
// 			queue = append(queue, child)
// 		}
// 	}
// }
