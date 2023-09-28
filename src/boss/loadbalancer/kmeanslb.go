package loadbalancer

import (
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
)

type Point []float64

func loadCentroids(filename string) ([]Point, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	byteValue, _ := ioutil.ReadAll(file)

	var centroids []Point
	json.Unmarshal(byteValue, &centroids)
	return centroids, nil
}

func assignToCluster(p Point, centroids []Point) int {
	minDist := math.MaxFloat64
	minIdx := 0
	for idx, centroid := range centroids {
		dist := distance(p, centroid)
		if dist < minDist {
			minDist = dist
			minIdx = idx
		}
	}
	return minIdx
}

func distance(p1, p2 Point) float64 {
	sum := 0.0
	for i := range p1 {
		delta := p1[i] - p2[i]
		sum += delta * delta
	}
	return math.Sqrt(sum)
}

func GetGroup(pkgs []float64) int {
	centroids, _ := loadCentroids("centroids.json")

	// Test the clustering with a new data point
	cluster := assignToCluster(pkgs, centroids)
	return cluster
}
