package docparser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var enumRegex = regexp.MustCompile(`enum=([\w ]+)`)

func parseFile(path string) (*ast.File, error) {
	data, err := ioutil.ReadFile(path) // just pass the file name
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet() // positions are relative to fset
	return parser.ParseFile(fset, "", data, parser.ParseComments)
}

type jsonTagInfo struct {
	name      string
	omitted   bool
	omitempty bool
	asString  bool
}

type validateTagInfo struct {
	required bool
	enum     []string
}

type jsonapiTagInfo struct {
	name        string
	isPrimary   bool
	primaryType string
	isAttribute bool
	isRelation  bool
	omitempty   bool
}

func haveTag(field *ast.Field, tagName string) (bool, error) {
	if field.Tag != nil && len(strings.TrimSpace(field.Tag.Value)) > 0 {
		tv, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			return false, err
		}

		if strings.TrimSpace(tv) != "" {
			st := reflect.StructTag(tv)

			return st.Get(tagName) != "", nil
		}
	}
	return false, nil
}

func parseJSONAPITag(field *ast.Field) (ja jsonapiTagInfo, err error) {
	if field.Tag != nil && len(strings.TrimSpace(field.Tag.Value)) > 0 {
		tv, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			return ja, err
		}

		if strings.TrimSpace(tv) != "" {
			st := reflect.StructTag(tv)

			jsonapiContent := st.Get("jsonapi")
			jsonapiData := strings.Split(jsonapiContent, ",")
			if len(jsonapiData) >= 2 {
				ja.isPrimary = jsonapiData[0] == "primary"
				ja.isAttribute = jsonapiData[0] == "attr"
				ja.isRelation = jsonapiData[0] == "relation"

				if !ja.isPrimary && !ja.isAttribute && !ja.isRelation {
					return ja, fmt.Errorf("can't be use a malformated tag (%s)", jsonapiContent)
				}

				if ja.isPrimary {
					// TODO: support json:"id"
					if strings.ToLower(field.Names[0].Name) == "id" {
						ja.name = "id"
						ja.primaryType = jsonapiData[1]
					} else {
						return ja, fmt.Errorf("can't have an primary field name without the name id")
					}
				} else {
					if jsonapiData[1] == "" {
						return ja, fmt.Errorf("can't have an empty field name (%s)", jsonapiContent)
					}
					ja.name = jsonapiData[1]
				}
			}
			if len(jsonapiData) > 2 && jsonapiData[2] == "omitempty" {
				if ja.isPrimary {
					return ja, fmt.Errorf("omitempty can't be used with primary (%s)", jsonapiContent)
				}
				ja.omitempty = true
			}

			return ja, nil
		}
	}
	return ja, nil
}

func parseJSONTag(field *ast.Field) (j jsonTagInfo, err error) {
	if len(field.Names) > 0 {
		j.name = field.Names[0].Name
	}
	if field.Tag != nil && len(strings.TrimSpace(field.Tag.Value)) > 0 {
		tv, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			return j, err
		}

		if strings.TrimSpace(tv) != "" {
			st := reflect.StructTag(tv)

			jsonTags := strings.Split(st.Get("json"), ",")
			jsonName := jsonTags[0]
			if jsonName == "-" {
				j.omitted = true
			} else if jsonName != "" {
				j.name = jsonName
			}

			if len(jsonTags) > 1 {
				j.omitempty = jsonTags[1] == "omitempty"
				j.asString = jsonTags[1] == "string"
			}

			return j, nil
		}
	}

	return j, nil
}

func parseValidateTag(field *ast.Field) (v validateTagInfo, err error) {
	if field.Tag != nil && len(strings.TrimSpace(field.Tag.Value)) > 0 {
		tv, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			return v, err
		}

		if strings.TrimSpace(tv) != "" {
			st := reflect.StructTag(tv)

			// https://github.com/go-playground/validator.v9
			validateData := strings.Split(st.Get("validate"), ",")

			required := false
			for _, vd := range validateData {
				if vd == "required" {
					required = true
				}
				if matches := enumRegex.FindStringSubmatch(vd); len(matches) > 0 {
					v.enum = strings.Fields(matches[1])
				}
			}
			v.required = required

			return v, nil
		}
	}
	return v, nil
}

func parseNamedType(gofile *ast.File, expr ast.Expr) (*schema, string, error) {
	p := schema{}
	switch ftpe := expr.(type) {
	case *ast.Ident: // simple value
		t, format, err := parseIdentProperty(ftpe)
		if err != nil {
			p.Ref = "#/components/schemas/" + t
			return &p, t, nil
		}
		p.Type = t
		p.Format = format
		return &p, t, nil
	case *ast.StarExpr: // pointer to something, optional by default
		p, t, _ := parseNamedType(gofile, ftpe.X)
		p.Nullable = true
		return p, t, nil
	case *ast.ArrayType: // slice type
		cp, t, _ := parseNamedType(gofile, ftpe.Elt)
		if cp.Format == "binary" {
			p.Type = "string"
			p.Format = "binary"
			return &p, t, nil
		}
		p.Type = "array"
		p.Items = make(map[string]itemData)
		if cp.Type != "" {
			p.Items["type"] = itemData{value: cp.Type}
		}
		if cp.Ref != "" {
			p.Items["$ref"] = itemData{value: cp.Ref}
		}
		return &p, t, nil
	case *ast.StructType:
		return nil, "", fmt.Errorf("expr (%+v) not yet unsupported", expr)
	case *ast.SelectorExpr:
		// @TODO ca va bugger ici !
		p, t, _ := parseNamedType(gofile, ftpe.X)
		return p, t, nil
	case *ast.MapType:
		k, _, kerr := parseNamedType(gofile, ftpe.Key)
		if kerr != nil {
			return nil, "", kerr
		}
		if k.Type != "string" {
			// keys can only be of type string
			return nil, "", fmt.Errorf("keys can only be of type string")
		}
		v, _, verr := parseNamedType(gofile, ftpe.Value)
		if verr != nil {
			return nil, "", verr
		}

		p.Type = "object"
		p.AdditionalProperties = v

		return &p, p.Type, nil
	case *ast.InterfaceType:
		return nil, "", fmt.Errorf("expr (%+v) not yet unsupported", expr)
	default:
		return nil, "", fmt.Errorf("expr (%+v) type (%+v) is unsupported for a schema", ftpe, expr)
	}
}

// https://swagger.io/specification/#dataTypes
func parseIdentProperty(expr *ast.Ident) (t, format string, err error) {
	switch expr.Name {
	case "string":
		t = "string"
	case "bson":
		t = "string"
	case "int":
		t = "integer"
	case "int8":
		t = "integer"
		format = "int8"
	case "int64":
		t = "integer"
		format = "int64"
	case "int32":
		t = "integer"
		format = "int32"
	case "time":
		t = "string"
		format = "date-time"
	case "float64":
		t = "number"
	case "bool":
		t = "boolean"
	case "byte":
		t = "string"
		format = "binary"
	default:
		t = expr.Name
		err = fmt.Errorf("Can't set the type %s", expr.Name)
	}
	return t, format, err
}
