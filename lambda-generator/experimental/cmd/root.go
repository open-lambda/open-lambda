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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tylerharter/open-lambda/lambda-generator/experimental/frontends"
	"github.com/tylerharter/open-lambda/lambda-generator/experimental/frontends/effe"
)

var (
	cfgFile     string
	frontendStr string
	fe          frontends.FrontEnd

	olDir string
)

// This represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   fmt.Sprintf("%s", os.Args[0]),
	Short: "Helps to template, organize, build and deploy OpenLambda Lambdas",
	Long:  "Helps to template, organize, build and deploy OpenLambda Lambdas",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports Persistent Flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.experimental.yaml)")
	RootCmd.PersistentFlags().StringVar(&frontendStr, "frontend", "effe", "OpenLambda frontend framework")
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// find the .openlambda folder or warn user if not found
	olDir = findOlDir()
	if olDir == "" {
		fmt.Printf("WARNING: no .openlambda directory found (Have you called %s init yet?)\n\n", os.Args[0])
		return
	}

	// Here we select the frontend, based on user configs found from above
	switch frontendStr {
	case "effe":
		fe = effe.NewFrontEnd(olDir)
	default:
		fmt.Println("frontend %s is unsupported\n")
		os.Exit(1)
	}
}

func findOlDir() string {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("failed to get wd with err %v\n", err)
		os.Exit(1)
	}

	curr, err := filepath.Abs(wd)
	if err != nil {
		fmt.Printf("failed to get create abs path with err %v\n", err)
		os.Exit(1)
	}

	// Walk one dir up each loop
	// TODO "/" is ommitted. Do we want this?
	for ; curr != "/"; curr = filepath.Dir(curr) {
		dir := filepath.Join(curr, ".openlambda")
		info, err := os.Stat(dir)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Printf("failed to get info on %s with err %v\n", dir, err)
				os.Exit(1)
			}
			continue
		}
		if !info.IsDir() {
			fmt.Printf("warning: non-directory .openlambda at %s\n", dir)
			continue
		}
		// found dir
		return dir
	}
	// caller will log
	return ""
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName("lambdagen") // name of config file (without extension)
	viper.AddConfigPath(olDir)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		// fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		err := writeConfig(filepath.Join(olDir, "lambdagen.json"), Config{
			DefaultFrontend: "effe",
			RegistryHost:    "localhost",
			RegistryPort:    "5000",
		})
		if err != nil {
			fmt.Printf("failed to write defailt config with err %v\n", err)
		}
	}
}

// This is kinda gross down here
type Config struct {
	DefaultFrontend string

	RegistryHost string
	RegistryPort string
}

func writeConfig(path string, conf Config) error {
	b, err := json.MarshalIndent(conf, "", "    ")
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString(string(b))
	return nil
}
