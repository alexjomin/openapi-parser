package docparser

import (
	"fmt"
	"go/ast"
)

func parseNamedType(gofile *ast.File, expr ast.Expr, sel *ast.Ident) (*schema, error) {
	p := schema{}
	switch ftpe := expr.(type) {
	case *ast.Ident: // simple value
		t, format, err := parseIdentProperty(ftpe, sel)
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
		t, err := parseNamedType(gofile, ftpe.X, sel)
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
		cp, err := parseNamedType(gofile, ftpe.Elt, sel)
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

			pnt, err := parseNamedType(gofile, field.Type, nil)
			if err != nil {
				return nil, err
			}

			p.Properties[j.name] = pnt

		}

		return &p, nil
	case *ast.SelectorExpr:
		t, err := parseNamedType(gofile, ftpe.X, ftpe.Sel)
		if err != nil {
			return nil, err
		}

		return t, nil
	case *ast.MapType:
		k, kerr := parseNamedType(gofile, ftpe.Key, sel)
		v, verr := parseNamedType(gofile, ftpe.Value, sel)
		if kerr != nil ||
			verr != nil ||
			(k.Type != "string"&&k.Type != "integer") {
			k, kerr := parseNamedType(gofile, ftpe.Key, sel);_,_=k, kerr
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
