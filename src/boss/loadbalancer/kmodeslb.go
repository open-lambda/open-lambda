package loadbalancer

import (
	"encoding/json"
	"io/ioutil"
	"math"
)

func hammingDistance(a, b []int) int {
	distance := 0
	for i := range a {
		if a[i] != b[i] {
			distance++
		}
	}
	return distance
}

func predictCluster(centroids [][]int, point []int) int {
	minDistance := math.MaxInt64
	cluster := -1
	for i, centroid := range centroids {
		distance := hammingDistance(centroid, point)
		if distance < minDistance {
			minDistance = distance
			cluster = i
		}
	}
	return cluster
}

func KModesGetGroup(pkgs []int) int {
	// Load centroids from JSON file
	data, err := ioutil.ReadFile("centroids_kmodes.json")
	if err != nil {
		panic(err)
	}

	var centroids [][]int
	err = json.Unmarshal(data, &centroids)
	if err != nil {
		panic(err)
	}

	// Predict cluster
	cluster := predictCluster(centroids, pkgs)
	return cluster
}
