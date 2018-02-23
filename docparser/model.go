package docparser

import (
	"go/ast"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var (
	regexpPath   = regexp.MustCompile("@openapi:path\n([^@]*)$")
	rexexpSchema = regexp.MustCompile(`@openapi:schema:?(\w+)?:?(?:\[([\w,]+)\])?`)
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
	spec.Components.Schemas = make(map[string]schema)
	return spec
}

type Components struct {
	Schemas         map[string]schema
	SecuritySchemes map[string]securitySchemes `yaml:"securitySchemes,omitempty"`
}

type securitySchemes struct {
	Type  string
	Flows map[string]flow
}

type flow struct {
	AuthorizationURL string            `yaml:"authorizationUrl"`
	TokenURL         string            `yaml:"tokenUrl"`
	Scopes           map[string]string `yaml:"scopes"`
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

func newEntity() schema {
	e := schema{}
	e.Properties = make(map[string]schema)
	e.Items = make(map[string]string)
	return e
}

type schema struct {
	Nullable             bool              `yaml:"nullable,omitempty"`
	Required             []string          `yaml:"required,omitempty"`
	Type                 string            `yaml:",omitempty"`
	Items                map[string]string `yaml:",omitempty"`
	Format               string            `yaml:"format,omitempty"`
	Ref                  string            `yaml:"$ref,omitempty"`
	Enum                 []string          `yaml:",omitempty"`
	Properties           map[string]schema `yaml:",omitempty"`
	AdditionalProperties bool              `yaml:"additionalProperties,omitempty"`
	OneOf                []schema          `yaml:"oneOf,omitempty"`
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
	RequestBody requestBody           `yaml:"requestBody,omitempty"`
	Security    []map[string][]string `yaml:",omitempty"`
	Headers     map[string]header     `yaml:",omitempty"`
}

type parameter struct {
	In          string
	Name        string
	Schema      schema `yaml:",omitempty"`
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
	Headers     map[string]header `yaml:",omitempty"`
}

type header struct {
	Description string `yaml:",omitempty"`
	Schema      schema `yaml:",omitempty"`
}

type content struct {
	Schema schema
}

func validatePath(path string) bool {
	// vendoring path
	if strings.Contains(path, "vendor") {
		return false
	}

	// not golang file
	if !strings.HasSuffix(path, ".go") {
		return false
	}

	// dot file
	if strings.HasPrefix(path, ".") {
		return false
	}

	return true
}

func (spec *openAPI) Parse(path string) {
	// fset := token.NewFileSet() // positions are relative to fset

	_ = filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if validatePath(path) {
			astFile, _ := parseFile(path)
			spec.parseSchemas(astFile)
			spec.parsePaths(astFile)
		}
		return nil
	})

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

		// TODO: Rafacto with parseNamedType
		for _, spc := range gd.Specs {

			// If the node is a Type
			if ts, ok := spc.(*ast.TypeSpec); ok {

				entityName := ts.Name.Name
				// fmt.Printf("type : %T %s\n", ts.Type, entityName)

				// Looking for openapi entity
				a := rexexpSchema.FindSubmatch([]byte(t))

				if len(a) == 0 {
					continue
				}

				// Name overide
				if len(a) == 2 {
					if string(a[1]) != "" {
						entityName = string(a[1])
					}
				}

				// array type
				if ar, ok := ts.Type.(*ast.ArrayType); ok {
					if i, ok := ar.Elt.(*ast.Ident); ok {
						e := newEntity()
						e.Type = "array"
						t, _, _ := parseIdentProperty(i)
						e.Items["type"] = t
						logrus.
							WithField("name", entityName).
							Info("Parsing Schema")
					}
				}

				// MapType
				if mp, ok := ts.Type.(*ast.MapType); ok {

					// only map[string]
					if i, ok := mp.Key.(*ast.Ident); ok {
						t, _, _ := parseIdentProperty(i)
						if t != "string" {
							continue
						}
					}

					e := newEntity()
					e.Type = "object"
					e.AdditionalProperties = true

					// map[string]interface{}
					if _, ok := mp.Value.(*ast.InterfaceType); ok {
						spec.Components.Schemas[entityName] = e
						logrus.
							WithField("name", entityName).
							Info("Parsing Schema")
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
							j, err := parseJSONTag(fld)
							if j.ignore {
								continue
							}
							p, err := parseNamedType(f, fld.Type)

							if j.required {
								e.Required = append(e.Required, j.name)
							}

							if err != nil {
								logrus.WithError(err).WithField("field", fld.Names[0]).Error("Can't parse the type of field in struct")
								continue
							}

							if len(j.enum) > 0 {
								p.Enum = j.enum
							}

							if p != nil {
								e.Properties[j.name] = *p
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

				// ArrayType
				if tpa, ok := ts.Type.(*ast.ArrayType); ok {
					entity := newEntity()
					p, err := parseNamedType(f, tpa.Elt)
					if err != nil {
						logrus.WithError(err).Error("Can't parse the type of field in struct")
						continue
					}

					entity.Type = "array"
					if p.Ref != "" {
						entity.Items["$ref"] = p.Ref
					} else {
						entity.Items["type"] = p.Type
					}

					spec.Components.Schemas[entityName] = entity
				}

				// if i, ok := ar.Elt.(*ast.Ident); ok {
				// 	e := newEntity()
				// 	e.Type = "array"
				// 	t, _, _ := parseIdentProperty(i)
				// 	e.Items["type"] = t
				// 	logrus.
				// 		WithField("name", entityName).
				// 		Info("Parsing Schema")
				// }

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
