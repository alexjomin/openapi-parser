package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/alexjomin/openapi-parser/docparser"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

var (
	outputPath   string
	inputPath    string
	parseVendors []string
	vendorsPath  string
	exitError    bool
)

// RootCmd represents the root command
var RootCmd = &cobra.Command{
	Use:   "openapi-parser",
	Short: "OpenAPI Parser ",
	Long:  `Parse comments in code to generate an OpenAPI documentation`,
	Run: func(cmd *cobra.Command, args []string) {
		spec := docparser.NewOpenAPI()
		spec.Parse(inputPath, parseVendors, vendorsPath, exitError)
		d, err := yaml.Marshal(&spec)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		_ = ioutil.WriteFile(outputPath, d, 0644)
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
	RootCmd.Flags().StringVar(&outputPath, "output", "openapi.yaml", "The output file")
	RootCmd.Flags().StringVar(&inputPath, "path", ".", "The Folder to parse")
	RootCmd.Flags().StringArrayVar(&parseVendors, "parse-vendors", []string{}, "Give the vendor to parse")
	RootCmd.Flags().StringVar(&vendorsPath, "vendors-path", "vendor", "Give the vendor path")
	RootCmd.Flags().BoolVar(&exitError, "exit-error", false, "When an error occurs on parsing, exit with a code > 0")
}
