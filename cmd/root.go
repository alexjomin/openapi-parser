// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
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
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/xee/parser-openapi/docparser"
	yaml "gopkg.in/yaml.v2"
)

var (
	output     string
	pathsDir   string
	schemasDir string
)

// RootCmd represents the root command
var RootCmd = &cobra.Command{
	Use:   "root",
	Short: "OpenAPI Parser ",
	Long:  `Parse comments in code to generate an OpenAPI documentation`,
	Run: func(cmd *cobra.Command, args []string) {

		spec := docparser.NewOpenAPI()

		if pathsDir != "" {
			files, err := ioutil.ReadDir(pathsDir)
			if err != nil {
				logrus.Fatal(err)
			}

			for _, lf := range files {
				if !strings.HasSuffix(lf.Name(), ".go") {
					continue
				}
				spec.ParsePathsFromFile(pathsDir + "/" + lf.Name())
			}
		}

		if schemasDir != "" {
			dirs := strings.Split(schemasDir, ",")
			for _, dir := range dirs {
				files, err := ioutil.ReadDir(dir)
				if err != nil {
					logrus.Fatal("error : ", err)
				}
				for _, lf := range files {
					if !strings.HasSuffix(lf.Name(), ".go") {
						continue
					}
					spec.ParseSchemasFromFile(dir + "/" + lf.Name())
				}
			}
		}

		d, err := yaml.Marshal(&spec)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		_ = ioutil.WriteFile(output, d, 0644)
	},
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
	RootCmd.PersistentFlags().StringVar(&output, "output", "openapi.yaml", "The output file")
	RootCmd.PersistentFlags().StringVar(&pathsDir, "paths", "", "The Handlers to parse")
	RootCmd.PersistentFlags().StringVar(&schemasDir, "schemas", "", "The Definitions struct to parse")
}
