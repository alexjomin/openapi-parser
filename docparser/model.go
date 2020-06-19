package docparser

import (
	"go/ast"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	regexpPath   = regexp.MustCompile("@openapi:path\n([^@]*)$")
	rexexpSchema = regexp.MustCompile(`@openapi:schema:?(\w+)?:?(?:\[([\w,]+)\])?`)
	regexpInfo   = regexp.MustCompile("@openapi:info\n([^@]*)$")
	tab          = regexp.MustCompile(`\t`)

	registeredSchemas = map[string]interface{}{
		"AnyValue": map[string]string{
			"description": "Can be anything: string, number, array, object, etc., including `null`",
		},
	}
)

type openAPI struct {
	Openapi    string
	Info       info
	Servers    []server
	Paths      map[string]path
	Tags       []tag `yaml:"tags,omitempty"`
	Components Components
	Security   []map[string][]string `yaml:"security,omitempty"`
	XGroupTags []interface{}         `yaml:"x-tagGroups"`
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
	spec.Components.Schemas = make(map[string]interface{})

	return spec
}

type Components struct {
	Schemas         map[string]interface{}     // schema or composedSchema
	SecuritySchemes map[string]securitySchemes `yaml:"securitySchemes,omitempty"`
}

type securitySchemes struct {
	Type   string
	Flows  map[string]flow `yaml:"flows,omitempty"`
	Scheme string          `yaml:"scheme,omitempty"`
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
	e.Properties = make(map[string]*schema)
	e.Items = make(map[string]interface{})
	return e
}

type metaSchema interface {
	RealName() string
	CustomName() string
	SetCustomName(string)
}

type metadata struct {
	RealName   string `yaml:"-"`
	CustomName string `yaml:"-"`
}

type composedSchema struct {
	metadata `yaml:"-"`
	AllOf    []*schema `yaml:"allOf"`
}

func (c *composedSchema) RealName() string {
	if c == nil {
		return ""
	}
	return c.metadata.RealName
}

func (c *composedSchema) CustomName() string {
	if c == nil {
		return ""
	}
	return c.metadata.CustomName
}

func (c *composedSchema) SetCustomName(customName string) {
	if c == nil {
		return
	}
	c.metadata.CustomName = customName
}

type schema struct {
	metadata             `yaml:"-"`
	Nullable             bool                   `yaml:"nullable,omitempty"`
	Required             []string               `yaml:"required,omitempty"`
	Type                 string                 `yaml:",omitempty"`
	Items                map[string]interface{} `yaml:",omitempty"`
	Format               string                 `yaml:"format,omitempty"`
	Ref                  string                 `yaml:"$ref,omitempty"`
	Enum                 []string               `yaml:",omitempty"`
	Properties           map[string]*schema     `yaml:",omitempty"`
	AdditionalProperties *schema                `yaml:"additionalProperties,omitempty"`
	OneOf                []schema               `yaml:"oneOf,omitempty"`
}

func (s *schema) RealName() string {
	if s == nil {
		return ""
	}
	return s.metadata.RealName
}

func (s *schema) CustomName() string {
	if s == nil {
		return ""
	}
	return s.metadata.CustomName
}

func (s *schema) SetCustomName(customName string) {
	if s == nil {
		return
	}
	s.metadata.CustomName = customName
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
	Example     string `yaml:"example,omitempty"`
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

func validatePath(path string, parseVendors []string) bool {
	// vendoring path
	if strings.Contains(path, "vendor") {
		found := false
		for _, vendorPath := range parseVendors {
			if strings.Contains(path, vendorPath) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
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

func (spec *openAPI) Parse(path string, parseVendors []string) {
	// fset := token.NewFileSet() // positions are relative to fset

	_ = filepath.Walk(path, func(path string, f os.FileInfo, err error) error {
		if validatePath(path, parseVendors) {
			astFile, _ := parseFile(path)
			spec.parseInfos(astFile)
			spec.parseSchemas(astFile)
			spec.parsePaths(astFile)
		}
		return nil
	})
	spec.composeSpecSchemas()
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

func replaceSchemaNameToCustom(s *schema) {
	if s == nil {
		return
	}

	for _, property := range s.Properties {
		replaceSchemaNameToCustom(property)
	}
	replaceSchemaNameToCustom(s.AdditionalProperties)

	refSplit := strings.Split(s.Ref, "/")
	if len(refSplit) != 4 {
		return
	}
	if replacementSchema, found := registeredSchemas[refSplit[3]]; found {
		meta, ok := replacementSchema.(metaSchema)
		if !ok {
			return
		}
		refSplit[3] = meta.CustomName()
	}
	s.Ref = strings.Join(refSplit, "/")
}

func (spec *openAPI) composeSpecSchemas() {
	for realName, registeredSchema := range registeredSchemas {
		if realName == "AnyValue" {
			spec.Components.Schemas[realName] = registeredSchema
			continue
		}

		meta, ok := registeredSchema.(metaSchema)
		if !ok {
			continue
		}

		if composed, ok := registeredSchema.(*composedSchema); ok {
			for _, s := range composed.AllOf {
				replaceSchemaNameToCustom(s)
			}
		} else if normal, ok := registeredSchema.(*schema); ok {
			replaceSchemaNameToCustom(normal)
		}

		name := realName
		if meta.CustomName() != "" {
			name = meta.CustomName()
		}
		spec.Components.Schemas[name] = registeredSchema
	}
}

func (spec *openAPI) parseMaps(mp *ast.MapType) *schema {
	// only map[string]
	if i, ok := mp.Key.(*ast.Ident); ok {
		t, _, _ := parseIdentProperty(i)
		if t != "string" {
			return nil
		}
	}

	e := newEntity()
	e.Type = "object"
	e.AdditionalProperties = &schema{}

	// map[string]interface{}
	if _, ok := mp.Value.(*ast.InterfaceType); ok {
		return &e
	}

	return nil
}

func (spec *openAPI) parseStructs(f *ast.File, tpe *ast.StructType) interface{} {
	var cs *composedSchema
	e := newEntity()
	e.Type = "object"

	for _, fld := range tpe.Fields.List {
		if len(fld.Names) > 0 && fld.Names[0] != nil && fld.Names[0].IsExported() {
			j, err := parseJSONTag(fld)
			if j.ignore {
				continue
			}
			p, err := parseNamedType(f, fld.Type, nil)

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
				e.Properties[j.name] = p
			}

		} else {
			// composition
			if cs == nil {
				cs = &composedSchema{
					AllOf: make([]*schema, 0),
				}
			}

			p, err := parseNamedType(f, fld.Type, nil)
			if err != nil {
				logrus.WithError(err).WithField("field", fld.Type).Error("Can't parse the type of composed field in struct")
				continue
			}

			cs.AllOf = append(cs.AllOf, p)
		}
	}

	if cs == nil {
		return &e
	} else {
		cs.AllOf = append(cs.AllOf, &e)
		return cs
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
				var entity interface{}
				realName := ts.Name.Name
				entityName := realName

				// Looking for openapi entity
				a := rexexpSchema.FindSubmatch([]byte(t))

				if len(a) == 0 {
					continue
				}

				if len(a) == 3 {
					if string(a[1]) != "" {
						entityName = string(a[1])
					}
				}

				switch n := ts.Type.(type) {
				case *ast.MapType:
					entity = spec.parseMaps(n)
					logrus.
						WithField("name", entityName).
						Info("Parsing Schema")

				case *ast.StructType:
					entity = spec.parseStructs(f, n)
					mtd, ok := entity.(metaSchema)
					if ok {
						mtd.SetCustomName(entityName)
					}
					logrus.
						WithField("name", entityName).
						Info("Parsing Schema")

				case *ast.ArrayType:
					e := newEntity()
					p, err := parseNamedType(f, n.Elt, nil)
					if err != nil {
						logrus.WithError(err).Error("Can't parse the type of field in struct")
						continue
					}

					e.Type = "array"
					if p.Ref != "" {
						e.Items["$ref"] = p.Ref
					} else {
						e.Items["type"] = p.Type
					}
					entity = &e

				default:
					p, err := parseNamedType(f, ts.Type, nil)
					if err != nil {
						logrus.WithError(err).Error("can't parse custom type")
						continue
					}
					p.SetCustomName(entityName)

					logrus.
						WithField("name", entityName).
						Info("Parsing Schema")
					entity = p
				}

				if entity != nil {
					registeredSchemas[realName] = entity
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

func (spec *openAPI) parseInfos(f *ast.File) {
	for _, s := range f.Comments {
		t := s.Text()
		// Test if comment is an info block
		a := regexpInfo.FindSubmatch([]byte(t))
		if len(a) == 0 {
			continue
		}

		// Replacing tab with spaces
		content := tab.ReplaceAllString(string(a[1]), "  ")

		// Unmarshal yaml
		infos := make(map[string]string)
		err := yaml.Unmarshal([]byte(content), &infos)
		if err != nil {
			logrus.
				WithError(err).
				WithField("content", content).
				Error("Unable to unmarshal infos")
			continue
		}

		version := infos["version"]
		if spec.Info.Version != "" && spec.Info.Version != version {
			logrus.
				WithField("version", spec.Info.Version).
				WithField("version_scanned", version).
				Warn("Version already exists and is different!")
		} else {
			logrus.
				WithField("field", "version").
				WithField("value", version).
				Info("Parsing info")
			spec.Info.Version = version
		}

		title := infos["title"]
		if spec.Info.Title != "" && spec.Info.Title != title {
			logrus.
				WithField("title", spec.Info.Title).
				WithField("title_scanned", title).
				Warn("Title already exists and is different!")
		} else {
			logrus.
				WithField("field", "title").
				WithField("value", title).
				Info("Parsing info")
			spec.Info.Title = title
		}

		description := infos["description"]
		if spec.Info.Description != "" && spec.Info.Description != description {
			logrus.
				WithField("description", spec.Info.Description).
				WithField("description_scanned", description).
				Warn("Description already exists and is different!")
		} else {
			logrus.
				WithField("field", "description").
				WithField("value", description).
				Info("Parsing info")
			spec.Info.Description = description
		}
	}
}
