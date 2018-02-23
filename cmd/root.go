package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/alexjomin/openapi-parser/docparser"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

var (
	output     string
	pathsDir   string
	schemasDir string
)

// RootCmd represents the root command
var RootCmd = &cobra.Command{
	Use:   "openapi-parser",
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
	RootCmd.Flags().StringVar(&output, "output", "openapi.yaml", "The output file")
	RootCmd.Flags().StringVar(&pathsDir, "paths", "", "The Handlers to parse")
	RootCmd.Flags().StringVar(&schemasDir, "schemas", "", "The Definitions struct to parse")
}
