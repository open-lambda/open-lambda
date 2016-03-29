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
)

var (
	targetDir string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init <target>",
	Short: "Initialize a template OpenLambda project in <target> directory",
	Long: `
Creates a .openlambda/ directory containing default project configurations.
`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// require a target init directory
		if len(args) < 1 {
			cmd.Help()
			os.Exit(1)
		}
		targetDir = args[0]
	},
	Run: func(cmd *cobra.Command, args []string) {

		if err := os.Chdir(targetDir); err != nil {
			fmt.Printf("failed to change dir to %s with err %v\n", targetDir, err)
			os.Exit(1)
		}
		wd, err := os.Getwd()
		if err != nil {
			fmt.Printf("failed to get wd with err %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Initializing OpenLambda at\n\t%s\n", wd)
		initManagementDir()
	},
}

func initManagementDir() {
	if err := os.Mkdir(".openlambda", 0777); err != nil {
		if os.IsExist(err) {
			fmt.Printf("OpenLambda project already exists at %s\n", targetDir)
			os.Exit(1)
		}
		fmt.Printf("failed to create .openlambda dir with err %v\n", err)
		os.Exit(1)
	}

	if err := os.Mkdir(".openlambda/frontends", 0777); err != nil {
		fmt.Printf("failed to create .openlambda/frontends dir with err %v\n", err)
		os.Exit(1)
	}

	// TODO: lay down configuration/tracking files
}

func init() {
	RootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
