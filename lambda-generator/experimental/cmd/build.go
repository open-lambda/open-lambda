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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build <optional file/directory ...>",
	Short: "Builds lambdas into OpenLambda compatable containers",
	Long: `
'build' will build all lambdas
'build <file>' will build the lambda in <file>
'build <directory>' will build all lambdas in directory
'build <directory> <file>' will build all lambdas in directory and lambda in file
Any combination of files and directories may be built.
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Printf("building all lambdas\n")
			root, err := os.Getwd()
			if err != nil {
				fmt.Printf("failed to get wd with err %v\n", err)
				os.Exit(1)
			}
			args = []string{root}
		}

		// validate all args
		// Place each in either map of files, or map of directories
		// Map is used for easy duplicate checking (bool unused)
		buildFiles := make(map[string]bool)
		buildDirs := make(map[string]bool)
		valid := true
		for _, path := range args {
			info, err := os.Stat(path)
			if err != nil {
				fmt.Printf("path '%s' seems funny (err: %v)\n", err)
				valid = false
				continue
			}

			if info.IsDir() {
				isDupAdd(buildDirs, path)
			} else {
				isDupAdd(buildFiles, path)
			}
		}
		if !valid {
			return
		}

		// Add any files in specified directories
		for path, _ := range buildDirs {
			for _, l := range getIndividualLambdas(path) {
				isDupAdd(buildFiles, l)
			}
		}

		// build each
		for path, _ := range buildFiles {
			// If we wanted polyglot projects, we need to detect lambda type here
			// Same goes for detecting support files vs lambdas...
			fmt.Printf("building '%s'\n", path)
			fe.BuildLambda(path)
		}
	},
}

func getIndividualLambdas(dir string) (lambdas []string) {
	lambdas = make([]string, 0)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("file %s caused error %v\n", path, err)
			return nil
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
		fmt.Printf("WARN: '%s' specified twice, will only be built once\n", key)
		return true
	}
	// value unused (false)
	m[key] = false
	return false
}

func init() {
	RootCmd.AddCommand(buildCmd)
}
