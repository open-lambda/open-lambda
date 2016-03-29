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

	"github.com/spf13/cobra"
	"github.com/tylerharter/open-lambda/lambda-generator/experimental/cmd/util"
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

		lambdas, err := util.FindLambdas(args)
		if err != nil {
			// Bad arguments
			return
		}

		// build each
		for _, path := range lambdas {
			// If we wanted polyglot projects, we need to detect lambda type here
			// Same goes for detecting support files vs lambdas...
			fmt.Printf("building '%s'\n", path)
			fe.BuildLambda(path)
		}
	},
}

func init() {
	RootCmd.AddCommand(buildCmd)
}
