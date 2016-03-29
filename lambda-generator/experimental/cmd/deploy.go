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
	"os/exec"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tylerharter/open-lambda/lambda-generator/experimental/cmd/util"
)

// deployCmd represents the deploy command
var deployCmd = &cobra.Command{
	Use:   "deploy <optional file/directory ...>",
	Short: "Deploys lambdas to the configured docker registry",
	Long: `
'deploy' will deploy all lambdas
'deploy <file>' will deploy the lambda in <file>
'deploy <directory>' will deploy all lambdas in directory
'deploy <directory> <file>' will deploy all lambdas in directory and lambda in file
Any combination of files and directories may be deployed.

The docker registry may be configured in '.openlambda/lambdagen.json'
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			fmt.Printf("Deploying all lambdas\n")
			root := path.Dir(olDir)
			args = []string{root}
		}

		lambdas, err := util.FindLambdas(args)
		if err != nil {
			// Bad arguments
			return
		}

		// deploy each
		for _, path := range lambdas {
			id, err := fe.GetId(path)
			if err != nil {
				fmt.Printf("failed to get Id for lambda %s with err %v\n", path, err)
				os.Exit(1)
			}
			host := viper.Get("RegistryHost")
			port := viper.Get("RegistryPort")
			img := fmt.Sprintf("%s:%s/%s", host, port, id)

			out, err := exec.Command("docker", "tag", id, img).Output()
			if err != nil {
				fmt.Printf("tag failed with output %s and err %v\n", out, err)
				os.Exit(1)
			}

			fmt.Printf("pushing %s\n", img)
			out, err = exec.Command("docker", "push", img).Output()
			if err != nil {
				fmt.Printf("push failed with output %s and err %v\n", out, err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	RootCmd.AddCommand(deployCmd)
}
