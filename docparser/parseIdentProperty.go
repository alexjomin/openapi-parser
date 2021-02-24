package docparser

import (
  "fmt"
  "go/ast"
)

// https://swagger.io/specification/#dataTypes
func parseIdentProperty(expr *ast.Ident, sel *ast.Ident) (t, format string, err error) {
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
    if nil != sel {
      name := expr.Name + "." + sel.Name
      if tp, ok := externalTypesMap[name]; ok {
        t = tp.Type
        format = tp.Format
        break
      } else if tp, ok := externalTypesMap[sel.Name]; ok {
        t = tp.Type
        format = tp.Format
        break
      } else if tp, ok := externalTypesMap[expr.Name]; ok {
        t = tp.Type
        format = tp.Format
        break
      }
    }
    //t = expr.Name
    err = fmt.Errorf("Can't set the type %s", expr.Name)
  }
  return t, format, err
}
