package docparser

import (
	"go/ast"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	regexpPath   = regexp.MustCompile("@openapi:path\n([^@]*)$")
	rexexpSchema = regexp.MustCompile(`@openapi:schema:?(\w+)?:?(?:\[([\w,]+)\])?`)
	regexpInfo   = regexp.MustCompile("@openapi:info\n([^@]*)$")
	tab          = regexp.MustCompile(`\t`)
)

type openAPI struct {
	Openapi                string
	Info                   info
	Servers                []server
	Paths                  map[string]path
	Tags                   []tag `yaml:"tags,omitempty"`
	Components             Components
	withoutJsonapiIncludes bool
}

type server struct {
	URL         string `yaml:"url"`
	Description string `yaml:"description"`
}

func NewOpenAPI(withoutJsonapiIncludes bool) openAPI {
	spec := openAPI{}
	spec.Openapi = "3.0.0"
	spec.Paths = make(map[string]path)
	spec.Components = Components{}
	spec.Components.Schemas = make(map[string]interface{})
	spec.withoutJsonapiIncludes = withoutJsonapiIncludes
	return spec
}

type Components struct {
	Schemas         map[string]interface{}     // schema or composedSchema
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

type itemData struct {
	value   string
	schemas itemDataSchema
}

type itemDataSchema []schema

func (v itemDataSchema) Len() int           { return len(v) }
func (v itemDataSchema) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v itemDataSchema) Less(i, j int) bool { return v[i].Ref < v[j].Ref }

func (a *itemData) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s []schema
	err := unmarshal(&a.schemas)
	if err != nil {
		var v string
		err := unmarshal(&a.value)
		if err != nil {
			return err
		}
		a = &itemData{
			value: v,
		}
	} else {
		a = &itemData{
			schemas: s,
		}
	}
	return nil
}

func (a itemData) MarshalYAML() (interface{}, error) {
	if a.schemas != nil {
		return a.schemas, nil
	}
	return a.value, nil
}

func newEntity() schema {
	e := schema{}
	e.Properties = make(map[string]schema)
	e.Items = make(map[string]itemData)
	return e
}

type composedSchema struct {
	AllOf []*schema `yaml:"allOf"`
}

type schema struct {
	Nullable             bool                `yaml:"nullable,omitempty"`
	Required             []string            `yaml:"required,omitempty"`
	Type                 string              `yaml:",omitempty"`
	Items                map[string]itemData `yaml:",omitempty"`
	Format               string              `yaml:"format,omitempty"`
	Ref                  string              `yaml:"$ref,omitempty"`
	Enum                 []string            `yaml:",omitempty"`
	Properties           map[string]schema   `yaml:",omitempty"`
	AdditionalProperties *schema             `yaml:"additionalProperties,omitempty"`
	OneOf                []schema            `yaml:"oneOf,omitempty"`
	AllOf                []schema            `yaml:"allOf,omitempty"`
	Discriminator        discriminator       `yaml:"discriminator,omitempty"`
}

type discriminator struct {
	PropertyName string            `yaml:"propertyName,omitempty"`
	Mapping      map[string]string `yaml:"mapping,omitempty"`
}

type items struct {
	Type string
}

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
			spec.parseInfos(astFile)
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
				Fatal("Unable to unmarshal path")
			return
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
							Fatal("Verb for this path already exists")
						return
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

	jsonapiIncludesResources := make(map[string][]string)

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		t := gd.Doc.Text()

		// TODO: Refacto with parseNamedType
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

				if len(a) == 3 {
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
						e.Items["type"] = itemData{value: t}
						logrus.
							WithField("name", entityName).
							Info("Parsing Schema of array")
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
					e.AdditionalProperties = &schema{}

					// map[string]interface{}
					if _, ok := mp.Value.(*ast.InterfaceType); ok {
						spec.Components.Schemas[entityName] = e
						logrus.
							WithField("name", entityName).
							Info("Parsing Schema of map")
					}
				}

				// StructType (with jsonapi tags)
				var isJSONAPIStruct bool
				if tpe, ok := ts.Type.(*ast.StructType); ok {
					var fieldsCount = 0
					var jsonapiFieldsCount = 0

					for _, fld := range tpe.Fields.List {
						if len(fld.Names) > 0 && fld.Names[0] != nil && fld.Names[0].IsExported() {
							fieldsCount++
							t, err := haveTag(fld, "jsonapi")
							if err != nil {
								logrus.WithError(err).WithField("name", entityName).
									Fatal("Can't used a malformed field in a jsonapi struct")
								return
							}
							if t {
								jsonapiFieldsCount++
							}
						}
					}

					if jsonapiFieldsCount > 0 {
						isJSONAPIStruct = true
						if fieldsCount != jsonapiFieldsCount {
							logrus.WithField("name", entityName).
								WithField("missing_fields", fieldsCount-jsonapiFieldsCount).
								Fatal("Can't used struct with missing jsonapi tags")
							return
						}
					}

					if isJSONAPIStruct {

						e := newEntity()
						e.Type = "object"
						e.Properties["data"] = schema{
							Ref: "#/components/schemas/" + entityName + "Data",
						}

						eID := newEntity()
						eID.Type = "object"
						eID.Properties["data"] = schema{
							Ref: "#/components/schemas/" + entityName + "IdentifierData",
						}

						collection := newEntity()
						collection.Type = "object"
						var dataField = schema{
							Type:  "array",
							Items: make(map[string]itemData),
						}
						dataField.Items["$ref"] = itemData{value: "#/components/schemas/" + entityName + "Data"}
						collection.Properties["data"] = dataField

						collectionID := newEntity()
						collectionID.Type = "object"
						var dataFieldID = schema{
							Type:  "array",
							Items: make(map[string]itemData),
						}
						dataFieldID.Items["$ref"] = itemData{value: "#/components/schemas/" + entityName + "IdentifierData"}
						collectionID.Properties["data"] = dataFieldID

						data := newEntity()
						data.Type = "object"
						data.Properties["attributes"] = schema{
							Ref: "#/components/schemas/" + entityName + "Attributes",
						}
						data.Properties["relationships"] = schema{
							Ref: "#/components/schemas/" + entityName + "Relationships",
						}

						dataID := newEntity()
						dataID.Type = "object"

						attributes := newEntity()
						attributes.Type = "object"

						relationships := newEntity()
						relationships.Type = "object"

						logrus.
							WithField("name", entityName).
							Info("Parsing Schema of Struct (with jsonapi tags)")

						for _, fld := range tpe.Fields.List {
							if len(fld.Names) > 0 && fld.Names[0] != nil && fld.Names[0].IsExported() {
								ja, err := parseJSONAPITag(fld)
								if err != nil {
									logrus.WithError(err).WithField("field", fld.Names[0]).
										Fatal("Can't parse jsonapi tag in field")
									return
								}

								if ja.isPrimary {
									addPrimaryFieldToJSONAPIData(&data, ja.primaryType)
									addPrimaryFieldToJSONAPIData(&dataID, ja.primaryType)
								} else {
									v, err := parseValidateTag(fld)
									if err != nil {
										logrus.WithError(err).WithField("field", fld.Names[0]).
											Fatal("Can't parse validate tag in field")
										return
									}

									var currentEntity = data
									if ja.isAttribute {
										currentEntity = attributes
									}
									if ja.isRelation {
										currentEntity = relationships
									}

									if v.required {
										currentEntity.Required = append(currentEntity.Required, ja.name)
									}

									p, t, err := parseNamedType(f, fld.Type)
									if err != nil {
										logrus.WithError(err).WithField("field", fld.Names[0]).
											Fatal("Can't parse the type of field in struct")
										return
									}

									if len(v.enum) > 0 {
										p.Enum = v.enum
									}

									if p != nil {
										if ja.isRelation {
											if p.Type != "array" {
												p.Ref += "Identifier"
											} else {
												np := &schema{
													Ref: p.Items["$ref"].value + "IdentifierCollection",
												}
												p = np
											}
											jsonapiIncludesResources[entityName] = append(jsonapiIncludesResources[entityName], t)
										}

										currentEntity.Properties[ja.name] = *p
									}
								}
							}
						}

						spec.Components.Schemas[entityName] = e
						spec.Components.Schemas[entityName+"Identifier"] = eID
						spec.Components.Schemas[entityName+"Collection"] = collection
						spec.Components.Schemas[entityName+"IdentifierCollection"] = collectionID
						spec.Components.Schemas[entityName+"Data"] = data
						spec.Components.Schemas[entityName+"IdentifierData"] = dataID
						spec.Components.Schemas[entityName+"Attributes"] = attributes
						spec.Components.Schemas[entityName+"Relationships"] = relationships
					}
				}

				// StructType (with json tags)
				if tpe, ok := ts.Type.(*ast.StructType); ok && !isJSONAPIStruct {
					var cs *composedSchema
					e := newEntity()
					e.Type = "object"

					logrus.
						WithField("name", entityName).
						Info("Parsing Schema of Struct (with json tags)")

					for _, fld := range tpe.Fields.List {
						if len(fld.Names) > 0 && fld.Names[0] != nil && fld.Names[0].IsExported() {
							v, err := parseValidateTag(fld)

							j, err := parseJSONTag(fld)

							if j.omitted {
								continue
							}

							p, _, err := parseNamedType(f, fld.Type)
							if err != nil {
								logrus.WithError(err).WithField("field", fld.Names[0]).Fatal("Can't parse the type of field in struct")
								return
							}

							if v.required {
								e.Required = append(e.Required, j.name)
							}

							if j.asString {
								// https://golang.org/pkg/encoding/json/#Marshal
								// The "string" option signals that a field is stored as JSON inside a JSON-encoded string.
								// It applies only to fields of string, floating point, integer, or boolean types.
								switch p.Type {
								case "string":
								case "integer", "number", "boolean":
									p.Format = p.Type
									p.Type = "string"
								default:
									logrus.WithError(err).WithField("field", fld.Names[0]).Fatal("Can't parse the type of field as authorized string format")
									return
								}
							}

							if len(v.enum) > 0 {
								p.Enum = v.enum
							}

							if p != nil {
								e.Properties[j.name] = *p
							}

						} else {
							// composition
							if cs == nil {
								cs = &composedSchema{
									AllOf: make([]*schema, 0),
								}
							}

							p, _, err := parseNamedType(f, fld.Type)
							if err != nil {
								logrus.WithError(err).WithField("field", fld.Type).Error("Can't parse the type of composed field in struct")
								continue
							}

							cs.AllOf = append(cs.AllOf, p)
						}
					}

					if cs == nil {
						spec.Components.Schemas[entityName] = e
					} else {
						cs.AllOf = append(cs.AllOf, &e)
						spec.Components.Schemas[entityName] = cs
					}
				}

				// ArrayType
				if tpa, ok := ts.Type.(*ast.ArrayType); ok {
					entity := newEntity()
					p, _, err := parseNamedType(f, tpa.Elt)
					if err != nil {
						logrus.WithError(err).Fatal("Can't parse the type of field in struct")
						return
					}

					entity.Type = "array"
					if p.Ref != "" {
						entity.Items["$ref"] = itemData{value: p.Ref}
					} else {
						entity.Items["type"] = itemData{value: p.Type}
					}

					spec.Components.Schemas[entityName] = entity
				}
			}
		}
	}

	if !spec.withoutJsonapiIncludes {
		for entityName := range jsonapiIncludesResources {
			relations := getTransitiveRelations(entityName, jsonapiIncludesResources, true)

			if len(relations) > 0 {
				e, ok := spec.Components.Schemas[entityName].(schema)
				if !ok {
					continue
				}
				if _, ok = e.Properties["includes"]; !ok {
					e.Properties["includes"] = schema{
						Nullable: true,
						Type:     "array",
						Items:    make(map[string]itemData),
					}
				}

				for _, relation := range relations {
					s := &schema{
						Ref: "#/components/schemas/" + relation + "Data",
					}

					includesAnyOf := e.Properties["includes"].Items["anyOf"]
					includesAnyOf.schemas = append(includesAnyOf.schemas, *s)
					sort.Sort(includesAnyOf.schemas)
					e.Properties["includes"].Items["anyOf"] = includesAnyOf
				}
			}
		}
	}
}

func getTransitiveRelations(entityName string, data map[string][]string, root bool) []string {
	relationsMap := make(map[string]bool)
	if !root {
		relationsMap[entityName] = true
	}
	if _, ok := data[entityName]; !ok || len(data[entityName]) != 0 {
		for _, v := range data[entityName] {
			if (v == entityName && root) || v != entityName {
				deps := getTransitiveRelations(v, data, false)
				for _, d := range deps {
					relationsMap[d] = true
				}
			}
		}
	}

	relations := make([]string, 0, len(relationsMap))
	for k := range relationsMap {
		relations = append(relations, k)
	}
	return relations
}

func appendOnce(a []string, b []string) []string {
	r := a
	for _, c := range b {
		e := false
		for _, d := range a {
			if c == d {
				e = true
			}
		}
		if !e {
			r = append(r, c)
		}
	}
	return r
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
