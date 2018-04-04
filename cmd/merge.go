package cmd

import (
	"io/ioutil"
	"log"
	"reflect"
	"strings"

	"github.com/alexjomin/openapi-parser/docparser"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
)

var (
	mainFile   string
	filesDir   string
	outputFile string
)

// mergeCmd represents the merge command
var mergeCmd = &cobra.Command{
	Use:   "merge",
	Short: "Merge multiple openapi specification into one",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		m, err := ioutil.ReadFile(mainFile)
		if err != nil {
			logrus.Fatal(err)
		}

		main := docparser.NewOpenAPI()
		err = yaml.Unmarshal(m, &main)
		if err != nil {
			logrus.Fatal(err)
		}

		files, err := ioutil.ReadDir(filesDir)
		if err != nil {
			logrus.Fatal(err)
		}

		logrus.Warn("Merging Files Frodon")

		for _, lf := range files {
			if !strings.HasSuffix(lf.Name(), ".yaml") {
				logrus.Warn("No YAML Files found")
				continue
			}
			m, err := ioutil.ReadFile(filesDir + "/" + lf.Name())
			if err != nil {
				logrus.Fatal(err)
			}

			spec := docparser.NewOpenAPI()
			err = yaml.Unmarshal(m, &spec)
			if err != nil {
				logrus.Fatal(err)
			}

			for k, v := range spec.Paths {
				url := k
				for verb, action := range v {
					logrus.WithField("verb", verb).WithField("url", url).Warn("Adding Path")
					main.AddAction(url, verb, action)
				}
			}

			for k, v := range spec.Components.Schemas {
				s, ok := main.Components.Schemas[k]
				if ok {
					result := reflect.DeepEqual(s, v)
					if !result {
						logrus.
							WithField("schema", k).
							WithField("file", lf.Name()).
							Fatal("Schema already exists and different !")
					}
					continue
				}
				main.Components.Schemas[k] = v
				logrus.WithField("schema", k).Warn("Adding Schema")
			}

		}

		d, err := yaml.Marshal(&main)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		_ = ioutil.WriteFile(outputFile, d, 0644)

	},
}

func init() {
	mergeCmd.Flags().StringVar(&mainFile, "main", "", "Path of the mainfile")
	mergeCmd.Flags().StringVar(&filesDir, "dir", "", "Path of the directory with the files you want to merge")
	mergeCmd.Flags().StringVar(&outputFile, "output", "merged-openapi.yaml", "Path of the result file")
	RootCmd.AddCommand(mergeCmd)
}
