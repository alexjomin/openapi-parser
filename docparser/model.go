package docparser

import (
	"fmt"
	"go/ast"
	"regexp"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var (
	regexpPath   = regexp.MustCompile("@openapi:path\n([^@]*)$")
	regexpEntity = regexp.MustCompile(`@openapi:schema:?(\w+)?`)
	tab          = regexp.MustCompile(`\t`)
)

type openAPI struct {
	Openapi    string
	Info       info
	Servers    []server
	Paths      map[string]path
	Tags       []tag `yaml:"tags,omitempty"`
	Components Components
}

type server struct {
	URL         string `yaml:"url"`
	Description string `yame:"description"`
}

func NewOpenAPI() openAPI {
	spec := openAPI{}
	spec.Openapi = "3.0.0"
	spec.Paths = make(map[string]path)
	spec.Components = Components{}
	spec.Components.Schemas = make(map[string]entity)
	return spec
}

type Components struct {
	Schemas map[string]entity
}

type info struct {
	Version     string
	Title       string
	Description string
	XLogo       map[string]string `yaml:"x-logo,omitempty"`
	Contact     map[string]string `yaml:",omitempty"`
	Licence     map[string]string `yaml:",omitempty"`
}

type tag struct {
	Name        string
	Description string
}

type entity struct {
	Type       string
	Required   []string            `yaml:",omitempty"`
	Items      map[string]string   `yaml:",omitempty"`
	Properties map[string]property `yaml:",omitempty"`
}

func newEntity() entity {
	e := entity{}
	e.Properties = make(map[string]property)
	e.Items = make(map[string]string)
	return e
}

type property struct {
	Nullable bool              `yaml:"nullable,omitempty"`
	Type     string            `yaml:",omitempty"`
	Items    map[string]string `yaml:",omitempty"`
	Format   string            `yaml:"format,omitempty"`
	Ref      string            `yaml:"$ref,omitempty"`
}

type items struct {
	Type string
}

// /pets: action
type path map[string]action

type action struct {
	Summary     string `yaml:",omitempty"`
	Description string
	Responses   map[string]response
	Tags        []string `yaml:",omitempty"`
	Parameters  []parameter
	RequestBody requestBody `yaml:"requestBody,omitempty"`
}

type parameter struct {
	In          string
	Name        string
	Schema      property `yaml:",omitempty"`
	Required    bool
	Description string
}

type requestBody struct {
	Description string
	Required    bool
	Content     map[string]content
}

type response struct {
	Content     map[string]content
	Description string
	Headers     map[string]header
}

type header struct {
	Description string   `yaml:",omitempty"`
	Schema      property `yaml:",omitempty"`
}

type content struct {
	Schema property
}

func (spec *openAPI) ParsePathsFromFile(file string) {
	logrus.WithField("name", file).Info("Parsing file")
	f, err := parseFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}
	spec.parsePaths(f)
}

func (spec *openAPI) ParseSchemasFromFile(file string) {
	logrus.WithField("name", file).Info("Parsing file")
	f, err := parseFile(file)
	if err != nil {
		fmt.Println(err)
		return
	}
	spec.parseSchemas(f)
}

func (spec *openAPI) parsePaths(f *ast.File) {
	for _, s := range f.Comments {
		t := s.Text()
		// Test if comments is a path
		a := regexpPath.FindSubmatch([]byte(t))
		if len(a) == 0 {
			continue
		}

		// Replacing tab with spaces
		content := tab.ReplaceAllString(string(a[1]), "  ")

		// Unmarshal yaml
		p := make(map[string]path)
		err := yaml.Unmarshal([]byte(content), &p)
		if err != nil {
			logrus.
				WithError(err).
				WithField("content", content).
				Error("Unable to unmarshal path")
			continue
		}

		for url, path := range p {
			// Path already exists in the spec
			if _, ok := spec.Paths[url]; ok {
				// Iterate over verbs
				for currentVerb, currentDesc := range path {
					if _, actionAlreadyExists := spec.Paths[url][currentVerb]; actionAlreadyExists {
						logrus.
							WithField("url", url).
							WithField("verb", currentVerb).
							Error("Verb for this path already exists")
						continue
					}
					spec.Paths[url][currentVerb] = currentDesc
				}
			} else {
				spec.Paths[url] = path
			}

			keys := []string{}
			for k := range path {
				keys = append(keys, k)
			}

			logrus.
				WithField("url", url).
				WithField("verb", keys).
				Info("Parsing path")
		}
	}
}

func (spec *openAPI) parseSchemas(f *ast.File) {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		t := gd.Doc.Text()
		for _, spc := range gd.Specs {

			// If the node is a Type
			if ts, ok := spc.(*ast.TypeSpec); ok {

				entityName := ts.Name.Name

				// Looking for openapi entity
				a := regexpEntity.FindSubmatch([]byte(t))

				if len(a) == 0 {
					continue
				}

				// Name overide
				if len(a) == 2 {
					if string(a[1]) != "" {
						entityName = string(a[1])
					}
				}

				// StructType
				if tpe, ok := ts.Type.(*ast.StructType); ok {
					e := newEntity()
					e.Type = "object"

					logrus.
						WithField("name", entityName).
						Info("Parsing Schema")

					for _, fld := range tpe.Fields.List {
						if len(fld.Names) > 0 && fld.Names[0] != nil && fld.Names[0].IsExported() {
							jsonname, ignore, required, _ := parseJSONTag(fld)
							if ignore {
								continue
							}
							p, err := parseNamedType(f, fld.Type)

							if required {
								e.Required = append(e.Required, jsonname)
							}

							if err != nil {
								logrus.WithError(err).WithField("field", fld.Names[0]).Error("Can't parse the type of field in struct")
								continue
							}
							if p != nil {
								e.Properties[jsonname] = *p
							}
							// @ToDO for composition
						} else {
							// switch t := fld.Type.(type) {
							// case *ast.Ident:
							// 	//fmt.Printf("indent : %v", t)
							// case *ast.SelectorExpr:
							// 	//fmt.Printf("SelectorExpr : %v", t)
							// case *ast.StarExpr:
							// 	//fmt.Printf("StarExpr : %v", t)
							// }
						}
					}
					spec.Components.Schemas[entityName] = e
				}

				if tpa, ok := ts.Type.(*ast.ArrayType); ok {
					entity := newEntity()
					p, err := parseNamedType(f, tpa.Elt)
					if err != nil {
						logrus.WithError(err).Error("Can't parse the type of field in struct")
						continue
					}
					entity.Type = "array"
					entity.Items["$ref"] = p.Ref
					spec.Components.Schemas[entityName] = entity
				}

			}
		}
	}
}

func (spec *openAPI) AddAction(path, verb string, a action) {
	if _, ok := spec.Paths[path]; !ok {
		spec.Paths[path] = make(map[string]action)
	}
	spec.Paths[path][verb] = a
}
