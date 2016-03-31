// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Given some paths of files and directories, returns the lambda paths
// Ignores .openlambda directories
func FindLambdas(paths []string) (files []string, err error) {
	// validate all args
	// Place each in either map of files, or map of directories
	// Map is used for easy duplicate checking (bool unused)
	filePaths := make(map[string]bool)
	dirPaths := make(map[string]bool)
	valid := true
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			fmt.Printf("path '%s' seems funny (err: %v)\n", err)
			valid = false
			continue
		}

		if info.IsDir() {
			isDupAdd(dirPaths, path)
		} else {
			isDupAdd(filePaths, path)
		}
	}
	if !valid {
		return nil, errors.New("invalid args given")
	}

	// Add any files in specified directories
	for path, _ := range dirPaths {
		for _, l := range getIndividualLambdas(path) {
			isDupAdd(filePaths, l)
		}
	}

	// convert to slice
	i := 0
	files = make([]string, len(filePaths))
	for f, _ := range filePaths {
		files[i] = f
		i++
	}

	return files, nil
}

// Given a dir, extracts lambdas
func getIndividualLambdas(dir string) (lambdas []string) {
	lambdas = make([]string, 0)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("file %s caused error %v\n", path, err)
			return nil
		}
		// Ignore .openlambda directories
		if info.IsDir() && info.Name() == ".openlambda" {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			lambdas = append(lambdas, path)
		}
		return nil
	})
	return lambdas
}

// Warn if duplicate, otherwise add
// Return true if dup
func isDupAdd(m map[string]bool, key string) bool {
	if _, dup := m[key]; dup {
		fmt.Printf("WARN: '%s' specified twice, will only be used once\n", key)
		return true
	}
	// value unused (false)
	m[key] = false
	return false
}
