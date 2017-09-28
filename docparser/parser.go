package docparser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"
)

func parseFile(path string) (*ast.File, error) {
	data, err := ioutil.ReadFile(path) // just pass the file name
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet() // positions are relative to fset
	return parser.ParseFile(fset, "", data, parser.ParseComments)
}

func parseJSONTag(field *ast.Field) (name string, ignore bool, err error) {
	if len(field.Names) > 0 {
		name = field.Names[0].Name
	}
	if field.Tag != nil && len(strings.TrimSpace(field.Tag.Value)) > 0 {
		tv, err := strconv.Unquote(field.Tag.Value)
		if err != nil {
			return name, false, err
		}

		if strings.TrimSpace(tv) != "" {
			st := reflect.StructTag(tv)
			jsonName := strings.Split(st.Get("json"), ",")[0]
			if jsonName == "-" {
				return name, true, nil
			} else if jsonName != "" {
				return jsonName, false, nil
			}
		}
	}
	return name, false, nil
}

func parseNamedType(gofile *ast.File, expr ast.Expr) (*property, error) {
	p := property{}
	switch ftpe := expr.(type) {
	case *ast.Ident: // simple value
		t, err := parseIdentProperty(ftpe)
		if err != nil {
			p.Ref = "#/components/schemas/" + t
			return &p, nil
		}
		p.Type = t
		return &p, nil
	case *ast.StarExpr: // pointer to something, optional by default
		// @TODO add something to handle nullable
		t, _ := parseNamedType(gofile, ftpe.X)
		t.Nullable = true
		return t, nil
	case *ast.ArrayType: // slice type
		cp, _ := parseNamedType(gofile, ftpe.Elt)
		p.Type = "array"
		p.Items = map[string]string{}
		p.Items["type"] = cp.Type
		return &p, nil
	case *ast.StructType:
		return nil, fmt.Errorf("expr (%s) not yet unsupported", expr)
	case *ast.SelectorExpr:
		// @TODO ca va bugger ici !
		t, _ := parseNamedType(gofile, ftpe.X)
		return t, nil
	case *ast.MapType:
		return nil, fmt.Errorf("expr (%s) not yet unsupported", expr)
	case *ast.InterfaceType:
		return nil, fmt.Errorf("expr (%s) not yet unsupported", expr)
	default:
		return nil, fmt.Errorf("expr (%s) type (%s) is unsupported for a schema", ftpe, expr)
	}
}

// https://swagger.io/specification/#dataTypes
func parseIdentProperty(expr *ast.Ident) (string, error) {
	switch expr.Name {
	case "string":
		return expr.Name, nil
	case "int":
		return "integer", nil
	case "int64":
		return "integer", nil
	case "int32":
		return "integer", nil
	case "time":
		return "string", nil
	case "float64":
		return "number", nil
	case "bool":
		return "boolean", nil
	}
	return expr.Name, fmt.Errorf("Can't set the type %s", expr.Name)
}
