package main

import (
	"fmt"

	"github.com/tylerharter/open-lambda/lambdaManager/dockerManager"
)

func main() {
	name := "hello-world"
	stdout, stderr, err := dockerManager.RunImg(name, nil)
	if err != nil {
		fmt.Printf("error in image run: %v\n", err)
	} else {
		fmt.Printf("stdout:\n\t  %v\n", stdout)
		fmt.Printf("stderr:\n\t  %v\n", stderr)
	}
}
