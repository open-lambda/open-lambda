package loadbalancer

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

const (
	Random   = 0
	KMeans   = 1
	KModes   = 2
	Sharding = 3
	Hash     = 4
)

var tree_path string

var MaxGroup int
var Lb *LoadBalancer
var Requirements map[string]string

type LoadBalancer struct {
	LbType int
}

func loadRequirements(root string) error {

	// Walk through the directory
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if it's a directory
		if info.IsDir() && path != root {
			requirementsPath := filepath.Join(path, "requirements.txt")

			// Read the contents of requirements.txt if it exists
			if _, err := os.Stat(requirementsPath); err == nil {
				content, err := ioutil.ReadFile(requirementsPath)
				if err != nil {
					return err
				}

				dirName := filepath.Base(path)
				Requirements[dirName] = string(content)
			}
		}
		return nil
	})

	return err
}

func InitLoadBalancer(lbType int, maxGroup int, path string) {
	tree_path = path
	if lbType != Random {
		// read requirements.txt into a data structure
		Requirements = make(map[string]string)
		err := loadRequirements("default-ol/registry/")
		if err != nil {
			log.Fatalf(err.Error())
		}
		if lbType == Sharding {
			fmt.Println("Debug")
			GetRoot()
			child1 := root.Children[0]
			fmt.Printf("%d\n", child1.SplitGeneration)
			if err != nil {
				log.Fatalf(err.Error())
			}
		}
		if lbType == Hash {
			initHasher()
		}
	}
	Lb = &LoadBalancer{
		LbType: lbType,
	}
	MaxGroup = maxGroup
}
