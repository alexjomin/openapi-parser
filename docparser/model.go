package docparser

import (
	"errors"
	"fmt"
	"go/ast"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	regexpPath    = regexp.MustCompile("@openapi:path\n([^@]*)$")
	regexpSchema  = regexp.MustCompile(`@openapi:schema:?(\w+)?:?(?:\[([\w,]+)\])?`)
	regexpExample = regexp.MustCompile(`@openapi:example [^\v\n]+`)
	regexpInfo    = regexp.MustCompile("@openapi:info\n([^@]*)$")
	regexpImport  = regexp.MustCompile(`import\(([^\)]+)\)`)
	tab           = regexp.MustCompile(`\t`)
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

	registeredSchemas map[string]interface{}
}

type server struct {
	URL         string                    `yaml:"url"`
	Description string                    `yaml:"description"`
	Variables   map[string]serverVariable `yaml:",omitempty"`
}

type serverVariable struct {
	Default     string
	Enum        []string
	Description string
}

func NewOpenAPI() openAPI {
	spec := openAPI{}
	spec.Openapi = "3.0.0"
	spec.Paths = make(map[string]path)
	spec.Components = Components{}
	spec.Components.Schemas = make(map[string]interface{})
	spec.registeredSchemas = map[string]interface{}{
		"AnyValue": map[string]string{
			"description": "Can be anything: string, number, array, object, etc., including `null`",
		},
	}
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

type externalDoc struct {
	Description string `yaml:",omitempty"`
	Url         string `yaml:"url,omitempty"`
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
	Nullable             *bool                  `yaml:"nullable,omitempty"`
	Required             []string               `yaml:"required,omitempty"`
	Type                 string                 `yaml:",omitempty"`
	Items                map[string]interface{} `yaml:",omitempty"`
	Format               string                 `yaml:"format,omitempty"`
	Ref                  string                 `yaml:"$ref,omitempty"`
	Enum                 []string               `yaml:",omitempty"`
	Properties           map[string]*schema     `yaml:",omitempty"`
	AdditionalProperties *schema                `yaml:"additionalProperties,omitempty"`
	OneOf                []schema               `yaml:"oneOf,omitempty"`
	Example              interface{}            `yaml:"example,omitempty"`
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

type BuildError struct {
	Err     error
	Content string
	Message string
}

func (e BuildError) Error() string {
	err, _ := logrus.
		WithError(e.Err).
		WithField("content", e.Content).
		WithField("message", e.Message).
		String()
	return err
}

// /pets: operation
type path map[string]operation

type operation struct {
	Summary      string `yaml:",omitempty"`
	Description  string
	ID           string `yaml:"operationId,omitempty"`
	Responses    map[string]response
	Tags         []string `yaml:",omitempty"`
	Parameters   []parameter
	RequestBody  requestBody           `yaml:"requestBody,omitempty"`
	Security     []map[string][]string `yaml:",omitempty"`
	Headers      map[string]header     `yaml:",omitempty"`
	Deprecated   bool                  `yaml:",omitempty"`
	Servers      []server              `yaml:",omitempty"`
	ExternalDocs externalDoc           `yaml:"externalDocs,omitempty"`
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

func (spec *openAPI) Parse(path string, parseVendors []string, vendorsPath string, exitNonZeroOnError bool) {
	// fset := token.NewFileSet() // positions are relative to fset

	walker := func(path string, f os.FileInfo, err error) error {
		if validatePath(path, parseVendors) {
			astFile, _ := parseFile(path)
			infosErrors := spec.parseInfos(astFile)
			schemasErrors := spec.parseSchemas(astFile)
			pathErrors := spec.parsePaths(astFile)
			if exitNonZeroOnError &&
				(len(infosErrors) > 0 || len(schemasErrors) > 0 || len(pathErrors) > 0) {
				return errors.New("errors while generating OpenAPI schema")
			}
		}
		return nil
	}

	err := filepath.Walk(path, walker)
	if err != nil {
		os.Exit(1)
	}

	err = filepath.Walk(vendorsPath, walker)
	if err != nil {
		os.Exit(1)
	}

	spec.composeSpecSchemas()
}

func (spec *openAPI) parsePaths(f *ast.File) (errs []error) {
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
			errs = append(errs, &BuildError{
				Err:     err,
				Content: content,
				Message: "unable to unmarshal path",
			})
			continue
		}

		for url, path := range p {
			// Path already exists in the spec
			if _, ok := spec.Paths[url]; ok {
				// Iterate over verbs
				for currentVerb, currentDesc := range path {
					if _, operationAlreadyExists := spec.Paths[url][currentVerb]; operationAlreadyExists {
						logrus.
							WithField("url", url).
							WithField("verb", currentVerb).
							Error("Verb for this path already exists")
						errs = append(errs, &BuildError{
							Err:     errors.New("verb for this path already exists"),
							Content: fmt.Sprintf("url: %s, verb: %s", url, currentVerb),
						})
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

	return
}

func (spec *openAPI) replaceSchemaNameToCustom(s *schema) {
	if s == nil {
		return
	}

	for _, property := range s.Properties {
		spec.replaceSchemaNameToCustom(property)
	}
	spec.replaceSchemaNameToCustom(s.AdditionalProperties)

	refSplit := strings.Split(s.Ref, "/")
	if len(refSplit) != 4 {
		return
	}
	if replacementSchema, found := spec.registeredSchemas[refSplit[3]]; found {
		meta, ok := replacementSchema.(metaSchema)
		if !ok {
			return
		}
		refSplit[3] = meta.CustomName()
	}
	s.Ref = strings.Join(refSplit, "/")
}

func (spec *openAPI) composeSpecSchemas() {
	for realName, registeredSchema := range spec.registeredSchemas {
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
				spec.replaceSchemaNameToCustom(s)
			}
		} else if normal, ok := registeredSchema.(*schema); ok {
			spec.replaceSchemaNameToCustom(normal)
		}

		name := realName
		if meta.CustomName() != "" {
			name = meta.CustomName()
		}
		spec.Components.Schemas[name] = registeredSchema
	}
}

func (spec *openAPI) parseMaps(mp *ast.MapType) (*schema, []error) {
	errors := make([]error, 0)

	// only map[string]
	if i, ok := mp.Key.(*ast.Ident); ok {
		t, _, err := parseIdentProperty(i)
		if err != nil {
			errors = append(errors, BuildError{
				Err:     err,
				Message: "could not parse ident property from map",
			})
		}

		if t != "string" {
			return nil, errors
		}
	}

	e := newEntity()
	e.Type = "object"
	e.AdditionalProperties = &schema{}

	// map[string]interface{}
	if _, ok := mp.Value.(*ast.InterfaceType); ok {
		return &e, errors
	}

	return nil, errors
}

func (spec *openAPI) parseStructs(f *ast.File, tpe *ast.StructType) (interface{}, []error) {
	errors := make([]error, 0)

	var cs *composedSchema
	e := newEntity()
	e.Type = "object"

	for _, fld := range tpe.Fields.List {

		example, err := spec.parseExample(fld.Doc.Text(), fld.Type)
		if err != nil {
			errors = append(errors, err)
		}

		if len(fld.Names) > 0 && fld.Names[0] != nil && fld.Names[0].IsExported() {
			j, err := parseJSONTag(fld)
			if j.ignore {
				continue
			}
			if j.required {
				e.Required = append(e.Required, j.name)
			}

			p, err := parseNamedType(f, fld.Type, nil)
			if err != nil {
				logrus.WithError(err).WithField("field", fld.Names[0]).Error("Can't parse the type of field in struct")
				errors = append(errors, BuildError{
					Err:     err,
					Content: fld.Names[0].String(),
					Message: "can't parse the type of field in struct",
				})
				continue
			}

			if example != nil {
				p.Example = example
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
				errors = append(errors, BuildError{
					Err:     err,
					Message: "can't parse the type of composed field in struct",
				})
				continue
			}

			if example != nil {
				p.Example = example
			}

			cs.AllOf = append(cs.AllOf, p)
		}
	}

	if cs == nil {
		return &e, errors
	} else {
		cs.AllOf = append(cs.AllOf, &e)
		return cs, errors
	}
}

func (spec *openAPI) parseExample(comment string, exampleType ast.Expr) (interface{}, error) {
	exampleLines := regexpExample.FindSubmatch([]byte(comment))
	if len(exampleLines) == 0 {
		return nil, nil
	}

	line := string(exampleLines[0])
	example := line[len("@openapi:example "):]

	return convertExample(example, exampleType)
}

func (spec *openAPI) parseSchemas(f *ast.File) (errors []error) {
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
				a := regexpSchema.FindSubmatch([]byte(t))
				example, err := spec.parseExample(t, ts.Type)
				if err != nil {
					errors = append(errors, err)
				}

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
					var errs []error
					entity, errs = spec.parseMaps(n)
					if len(errs) != 0 {
						errors = append(errors, errs...)
					}

					logrus.
						WithField("name", entityName).
						Info("Parsing Schema")

				case *ast.StructType:
					var errs []error
					entity, errs = spec.parseStructs(f, n)
					if len(errs) != 0 {
						errors = append(errors, errs...)
					}

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
						errors = append(errors, &BuildError{
							Err:     err,
							Message: "Can't parse the type of field in struct",
						})
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
						errors = append(errors, BuildError{
							Err:     err,
							Message: "can't parse custom type",
						})
						continue
					}
					p.SetCustomName(entityName)

					logrus.
						WithField("name", entityName).
						Info("Parsing Schema")
					entity = p
				}

				if entity != nil {
					s, ok := entity.(*schema)
					if ok && example != nil {
						s.Example = example
					}
					spec.registeredSchemas[realName] = entity
				}
			}
		}
	}
	return
}

func (spec *openAPI) AddOperation(path, verb string, a operation) {
	if _, ok := spec.Paths[path]; !ok {
		spec.Paths[path] = make(map[string]operation)
	}
	spec.Paths[path][verb] = a
}

func (spec *openAPI) parseInfos(f *ast.File) (errors []error) {
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
		infos := info{}
		err := yaml.Unmarshal([]byte(content), &infos)
		if err != nil {
			logrus.
				WithError(err).
				WithField("content", content).
				Error("Unable to unmarshal infos")
			errors = append(errors, &BuildError{
				Err:     err,
				Content: content,
				Message: "Unable to unmarshal infos",
			})
			continue
		}

		version := infos.Version
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

		title := infos.Title
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

		description := infos.Description
		if spec.Info.Description != "" && spec.Info.Description != description {
			logrus.
				WithField("description", spec.Info.Description).
				WithField("description_scanned", description).
				Warn("Description already exists and is different!")
		} else {
			p, err := parseImportContentPath(description)
			// no need to import a file
			if err != nil {
				logrus.
					WithField("field", "description").
					WithField("value", description).
					Info("Parsing info")
				spec.Info.Description = description
			} else {
				c, err := ioutil.ReadFile(p)

				if err != nil {
					logrus.
						WithField("File", p).
						WithError(err).
						Error("Could not import file")
					return
				}

				logrus.
					WithField("field", "description").
					WithField("value", "content of file: "+p).
					Info("Parsing info")
				spec.Info.Description = string(c)
			}
		}

		spec.Info.XLogo = infos.XLogo

	}
	return
}

func parseImportContentPath(str string) (string, error) {
	matches := regexpImport.FindStringSubmatch(str)
	if len(matches) == 2 {
		return matches[1], nil
	}
	return "", errors.New("Not an import")
}
