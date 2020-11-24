package docparser

import (
	"fmt"
	"go/ast"
	"strconv"
)

func convertExample(example string, exampleType ast.Expr) (interface{}, error) {
	expr, ok := exampleType.(*ast.Ident)
	if !ok {
		return example, nil
	}

	switch expr.Name {
	case "int", "int8", "int32", "int64":
		i, err := strconv.ParseInt(example, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse int: %w", err)
		}
		return i, nil

	case "uint", "uint8", "uint32", "uint64":
		u, err := strconv.ParseUint(example, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse uint: %w", err)
		}

		return u, nil

	case "float", "float32", "float64":
		f, err := strconv.ParseFloat(example, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse float: %w", err)
		}

		return f, nil

	default:
		return example, nil
	}
}
