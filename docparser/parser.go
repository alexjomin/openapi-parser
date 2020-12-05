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
var oneOfRegex = regexp.MustCompile(`oneof=([\w ]+)`) // validator.v9 enum tag is oneof

func parseFile(path string) (*ast.File, error) {
	data, err := ioutil.ReadFile(path) // just pass the file name
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet() // positions are relative to fset
	return parser.ParseFile(fset, "", data, parser.ParseComments)
}

type jsonTagInfo struct {
	name     string
	ignore   bool
	required bool
	enum     []string
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

			jsonName := strings.Split(st.Get("json"), ",")[0]
			if jsonName == "-" {
				j.ignore = true
				j.required = false
				return j, nil
			} else if jsonName != "" {
				required := false
				// https://github.com/go-playground/validator
				// check if validate attr is active
				validateData := strings.Split(st.Get("validate"), ",")
				for _, v := range validateData {
					if v == "required" {
						required = true
					}
					if matches := enumRegex.FindStringSubmatch(v); len(matches) > 0 {
						j.enum = strings.Fields(matches[1])
					}
					if matches := oneOfRegex.FindStringSubmatch(v); len(matches) > 0 {
						j.enum = strings.Fields(matches[1])
					}
				}

				j.name = jsonName
				j.required = required
				j.ignore = false

				return j, nil
			}
		}
	}
	return j, nil
}

func parseNamedType(expr ast.Expr, sel *ast.Ident) (*schema, error) {
	p := schema{}
	switch ftpe := expr.(type) {
	case *ast.Ident: // simple value
		t, format, err := parseIdentProperty(ftpe)
		if err != nil {
			p.Ref = "#/components/schemas/"
			if sel != nil {
				p.Ref += sel.Name
				p.metadata.RealName = sel.Name
			} else {
				p.Ref += t
				p.metadata.RealName = t
			}
			return &p, nil
		}
		p.Type = t
		p.Format = format
		return &p, nil
	case *ast.StarExpr: // pointer to something, optional by default
		t, err := parseNamedType(ftpe.X, sel)
		if err != nil {
			return nil, err
		}
		if t.Ref == "" {
			// if ref, cannot have other properties
			tBool := true
			t.Nullable = &tBool
		}
		return t, nil
	case *ast.ArrayType: // slice type
		cp, err := parseNamedType(ftpe.Elt, sel)
		if err != nil {
			return nil, err
		}

		if cp.Format == "binary" {
			p.Type = "string"
			p.Format = "binary"
			return &p, nil
		}
		p.Type = "array"
		p.Items = map[string]interface{}{}
		if cp.Type != "" {
			p.Items["type"] = cp.Type
			if len(cp.Items) != 0 {
				p.Items["items"] = cp.Items
			}
			if len(cp.Properties) != 0 {
				p.Items["properties"] = cp.Properties
			}
		}
		if cp.Ref != "" {
			p.Items["$ref"] = cp.Ref
		}
		return &p, nil
	case *ast.StructType:
		p = newEntity()
		p.Type = "object"

		for _, field := range ftpe.Fields.List {
			j, err := parseJSONTag(ftpe.Fields.List[0])
			if err != nil {
				return nil, err
			}

			pnt, err := parseNamedType(field.Type, nil)
			if err != nil {
				return nil, err
			}

			p.Properties[j.name] = pnt

		}

		return &p, nil
	case *ast.SelectorExpr:
		t, err := parseNamedType(ftpe.X, ftpe.Sel)
		if err != nil {
			return nil, err
		}

		return t, nil
	case *ast.MapType:
		k, kerr := parseNamedType(ftpe.Key, sel)
		v, verr := parseNamedType(ftpe.Value, sel)
		if kerr != nil || verr != nil || k.Type != "string" {
			// keys can only be of type string
			return nil, fmt.Errorf("expr (%s) not yet unsupported", expr)
		}

		p.Type = "object"
		p.AdditionalProperties = v

		return &p, nil
	case *ast.InterfaceType:
		p.Ref = "#/components/schemas/AnyValue"
		return &p, nil
	default:
		return nil, fmt.Errorf("expr (%s) type (%s) is unsupported for a schema", ftpe, expr)
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
	case "byte", "json":
		t = "string"
		format = "binary"
	default:
		t = expr.Name
		err = fmt.Errorf("Can't set the type %s", expr.Name)
	}
	return t, format, err
}
