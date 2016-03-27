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
	"os"

	"github.com/spf13/cobra"
)

var (
	location string
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add <location>",
	Short: "Adds a template lambda at <location>",
	Long: `Adds a new template lambda file at <location> creating any neccesary directories.
Example (using effe frontend):
	` + os.Args[0] + ` add my/new/handler
Would create directories 'my/' and 'my/new/' along with file 'handler.go'
'handler.go' would contain an effe template frontend.`,
	PreRun: func(cmd *cobra.Command, args []string) {
		// require a location
		if len(args) < 1 {
			cmd.Help()
			os.Exit(1)
		}
		location = args[0]
	},
	Run: func(cmd *cobra.Command, args []string) {
		if fe != nil {
			fe.AddLambda(location)
		}
	},
}

func init() {
	RootCmd.AddCommand(addCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// addCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// addCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

}
