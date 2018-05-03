package docparser

import (
	"go/ast"
	"go/token"
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"
)

type parseJSONTagTestCase struct {
	description     string
	field           *ast.Field
	expectedJSONTag jsonTagInfo
	expectedError   string
}

type parseValidateTagTestCase struct {
	description         string
	field               *ast.Field
	expectedValidateTag validateTagInfo
	expectedError       string
}

type parseIdentPropertyTestCase struct {
	description    string
	expr           *ast.Ident
	expectedType   string
	expectedError  string
	expectedFormat string
}
type parseNamedTypeTestCase struct {
	description    string
	gofile         *ast.File
	expr           ast.Expr
	expectedSchema *schema
	expectedError  string
}
type parseFileTestCase struct {
	description         string
	goFilePath          string
	expectedFilePackage token.Pos
	expectedError       string
	expectedFileName    string
}

func TestParseFile(t *testing.T) {
	testCases := []parseFileTestCase{
		{
			description:   "should throw err with no path",
			expectedError: "open : no such file or directory",
		},
		{
			description:         "should parse incorrect file",
			goFilePath:          "../Makefile",
			expectedError:       "1:1: expected 'package', found '.' (and 1 more errors)",
			expectedFilePackage: 0,
		},
		{
			description:         "should parse file",
			goFilePath:          "../datatest/user/components.go",
			expectedFilePackage: 1,
			expectedFileName:    "user",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			file, err := parseFile(tc.goFilePath)
			if len(tc.expectedError) > 0 {
				if (err != nil) && (err.Error() != tc.expectedError) {
					t.Errorf("got error: %v, wantErr: %v", err, tc.expectedError)
				}
				if err == nil {
					t.Fatalf("expected error: %v . Got nothing", tc.expectedError)
				}
			}
			if (err != nil) && (len(tc.expectedError) == 0) {
				t.Fatalf("unexpected error: %v", err)
			}
			if file != nil && file.Name != nil && file.Name.Name != tc.expectedFileName {
				t.Errorf("got: %v, want: %v", file.Name, tc.expectedFileName)
			}
			if file != nil && file.Package != tc.expectedFilePackage {
				t.Errorf("got: %v, want: %v", file.Name, tc.expectedFileName)
			}
		})
	}
}

func TestParseNamedType(t *testing.T) {
	testCases := []parseNamedTypeTestCase{
		{
			description:    "Should parse *ast.Ident with unknown name",
			expr:           &ast.Ident{Name: "unknown"},
			expectedSchema: &schema{Ref: "#/components/schemas/unknown"},
		},

		{
			description:    "Should parse *ast.Ident with name string",
			expr:           &ast.Ident{Name: "string"},
			expectedSchema: &schema{Type: "string"},
		},
		{
			description:    "Should parse *ast.Ident with name time",
			expr:           &ast.Ident{Name: "time"},
			expectedSchema: &schema{Type: "string", Format: "date-time"},
		},
		{
			description:    "Should parse *ast.StarExpr and set Nullable",
			expr:           &ast.StarExpr{X: &ast.Ident{Name: "time"}},
			expectedSchema: &schema{Type: "string", Format: "date-time", Nullable: true},
		},
		{
			description: "Should parse *ast.ArrayType with know type",
			expr:        &ast.ArrayType{Elt: &ast.Ident{Name: "time"}},
			expectedSchema: &schema{Type: "array", Items: map[string]itemData{
				"type": itemData{value: "string"},
			}},
		},
		{
			description: "Should parse *ast.ArrayType with unknown type",
			expr:        &ast.ArrayType{Elt: &ast.Ident{Name: "unknown"}},
			expectedSchema: &schema{Type: "array", Items: map[string]itemData{
				"$ref": itemData{value: "#/components/schemas/unknown"},
			}},
		},
		{
			description:   "Should throw error when parse *ast.StructType",
			expr:          &ast.StructType{},
			expectedError: "expr (&{Struct:0 Fields:<nil> Incomplete:false}) not yet unsupported",
		},
		{
			description: "Should throw error when parse *ast.MapType[nil]nil",
			expr: &ast.MapType{
				Key:   nil,
				Value: nil,
			},
			expectedError: "expr (<nil>) type (<nil>) is unsupported for a schema",
		},
		{
			description: "Should throw error when parse *ast.MapType[string]interface{}",
			expr: &ast.MapType{
				Key:   &ast.Ident{Name: "string"},
				Value: &ast.InterfaceType{},
			},
			expectedError: "expr (&{Interface:0 Methods:<nil> Incomplete:false}) not yet unsupported",
		},
		{
			description: "Should parse *ast.MapType[string]string",
			expr: &ast.MapType{
				Key:   &ast.Ident{Name: "string"},
				Value: &ast.Ident{Name: "string"},
			},
			expectedSchema: &schema{
				Type:                 "object",
				AdditionalProperties: &schema{Type: "string"},
			},
		},
		{
			description: "Should parse *ast.MapType[string]Pet",
			expr: &ast.MapType{
				Key:   &ast.Ident{Name: "string"},
				Value: &ast.Ident{Name: "Pet"},
			},
			expectedSchema: &schema{
				Type: "object",
				AdditionalProperties: &schema{
					Ref: "#/components/schemas/Pet",
				},
			},
		},
		{
			description: "Should throw error when parse *ast.MapType[Object]Pet",
			expr: &ast.MapType{
				Key:   &ast.Ident{Name: "Object"},
				Value: &ast.Ident{Name: "Pet"},
			},
			expectedError: "keys can only be of type string",
		},
		{
			description:   "Should throw error when parse *ast.InterfaceType",
			expr:          &ast.InterfaceType{},
			expectedError: "expr (&{Interface:0 Methods:<nil> Incomplete:false}) not yet unsupported",
		},
		{
			description:    "Should parse *ast.SelectorExpr",
			expr:           &ast.SelectorExpr{X: &ast.Ident{Name: "time"}},
			expectedSchema: &schema{Type: "string", Format: "date-time"},
		},

		{
			description: "Should parse correctly a selector of an array of pointer of unknown type",
			expr:        &ast.SelectorExpr{X: &ast.ArrayType{Elt: &ast.StarExpr{X: &ast.Ident{Name: "unknown"}}}},
			expectedSchema: &schema{Type: "array", Items: map[string]itemData{
				"$ref": itemData{value: "#/components/schemas/unknown"},
			}},
		},
		{
			description: "Should parse correctly a selector of an array of pointer of time type",
			expr:        &ast.SelectorExpr{X: &ast.ArrayType{Elt: &ast.StarExpr{X: &ast.Ident{Name: "time"}}}},
			expectedSchema: &schema{Type: "array", Items: map[string]itemData{
				"type": itemData{value: "string"},
			}},
		},
		{
			description:   "Should return  error for unsupported types",
			expr:          &ast.FuncType{},
			expectedError: "expr (&{Func:0 Params:<nil> Results:<nil>}) type (&{Func:0 Params:<nil> Results:<nil>}) is unsupported for a schema",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			schema, _, err := parseNamedType(tc.gofile, tc.expr)
			if len(tc.expectedError) > 0 {
				if (err != nil) && (err.Error() != tc.expectedError) {
					t.Errorf("got error: %v, wantErr: %v", err, tc.expectedError)
				}
				if err == nil {
					t.Fatalf("expected error: %v . Got nothing", tc.expectedError)
				}
			}
			if (err != nil) && (len(tc.expectedError) == 0) {
				t.Fatalf("unexpected error: %v", err)
			}

			bSchema, serr := yaml.Marshal(&schema)
			bExpectedSchema, eserr := yaml.Marshal(&tc.expectedSchema)
			if serr != nil || eserr != nil {
				t.Error(err)
			}
			if !reflect.DeepEqual(bSchema, bExpectedSchema) {
				t.Errorf("got: %+v, want: %+v\n", schema, tc.expectedSchema)
				t.Errorf("got: %+v, want: %+v\n", schema.AdditionalProperties, tc.expectedSchema.AdditionalProperties)
			}
		})
	}
}

func TestParseJSONTag(t *testing.T) {
	testCases := []parseJSONTagTestCase{
		{
			description: "Should parse json tag value with value 'jsontagname,omitempty'",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`json:\"jsontagname,omitempty\"`",
				},
			},
			expectedJSONTag: jsonTagInfo{
				name:      "jsontagname",
				omitempty: true,
			},
		},
		{
			description: "Should parse json tag value with value ',omitempty'",
			field: &ast.Field{
				Names: []*ast.Ident{
					&ast.Ident{
						Name: "jsonTagName",
					},
				},
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`json:\",omitempty\"`",
				},
			},
			expectedJSONTag: jsonTagInfo{
				name:      "jsonTagName",
				omitempty: true,
			},
		},
		{
			description: "Should parse json tag value with value ',string'",
			field: &ast.Field{
				Names: []*ast.Ident{
					&ast.Ident{
						Name: "jsonTagName",
					},
				},
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`json:\",string\"`",
				},
			},
			expectedJSONTag: jsonTagInfo{
				name:     "jsonTagName",
				asString: true,
			},
		},
		{
			description: "Should fail with invalid syntax in tag value",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "Test",
				},
			},
			expectedJSONTag: jsonTagInfo{},
			expectedError:   "invalid syntax",
		}, {
			description:     "Should not set name in jsontag",
			field:           &ast.Field{},
			expectedJSONTag: jsonTagInfo{},
		},
		{
			description: "Should set name in jsontag",
			field: &ast.Field{
				Names: []*ast.Ident{
					&ast.Ident{
						Name: "testName",
					},
				}},
			expectedJSONTag: jsonTagInfo{
				name: "testName"},
		},
		{
			description: "Should set firstname in jsontag",
			field: &ast.Field{
				Names: []*ast.Ident{
					&ast.Ident{
						Name: "firstTestName",
					},
					&ast.Ident{
						Name: "secondTestName",
					},
				}},
			expectedJSONTag: jsonTagInfo{
				name: "firstTestName"},
		},
		{
			description: "Should return empty jsonTagInfo",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "",
				},
			},
			expectedJSONTag: jsonTagInfo{},
		},
		{
			description: "Should parse tag value",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`Test`",
				},
			},
			expectedJSONTag: jsonTagInfo{},
		},
		{
			description: "Should parse json tag value with value -  ",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`json:\"-\"`",
				},
			},
			expectedJSONTag: jsonTagInfo{
				omitted: true,
			},
		},
		{
			description: "Should parse json tag value with value jsontagname",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`json:\"jsontagname\"`",
				},
			},
			expectedJSONTag: jsonTagInfo{
				name: "jsontagname",
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			parsedJSONTag, err := parseJSONTag(tc.field)
			if len(tc.expectedError) > 0 {
				if (err != nil) && (err.Error() != tc.expectedError) {
					t.Errorf("got error: %v, wantErr: %v", err, tc.expectedError)
				}
				if err == nil {
					t.Fatalf("expected error: %v . Got nothing", tc.expectedError)
				}
			}
			if (err != nil) && (len(tc.expectedError) == 0) {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tc.expectedJSONTag, parsedJSONTag) {
				t.Errorf("got: %v, want: %v", parsedJSONTag, tc.expectedJSONTag)
			}
		})
	}
}

func TestParseValidateTag(t *testing.T) {
	testCases := []parseValidateTagTestCase{
		{
			description: "Should read required validation",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`json:\"jsontagname\" validate:\"required\"`",
				},
			},
			expectedValidateTag: validateTagInfo{
				required: true,
			},
		},
		{
			description: "Should read enum validation",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`json:\"jsontagname\" validate:\"required,enum=a b\"`",
				},
			},
			expectedValidateTag: validateTagInfo{
				required: true,
				enum:     []string{"a", "b"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			parsedValidateTag, err := parseValidateTag(tc.field)
			if len(tc.expectedError) > 0 {
				if (err != nil) && (err.Error() != tc.expectedError) {
					t.Errorf("got error: %v, wantErr: %v", err, tc.expectedError)
				}
				if err == nil {
					t.Fatalf("expected error: %v . Got nothing", tc.expectedError)
				}
			}
			if (err != nil) && (len(tc.expectedError) == 0) {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(tc.expectedValidateTag, parsedValidateTag) {
				t.Errorf("got: %v, want: %v", parsedValidateTag, tc.expectedValidateTag)
			}
		})
	}
}

func TestParseIdentProperty(t *testing.T) {
	testCases := []parseIdentPropertyTestCase{
		{
			description:   "parse empty ident",
			expr:          &ast.Ident{},
			expectedError: "Can't set the type ",
		},
		{
			description:   "parse unknown ident type",
			expr:          &ast.Ident{Name: "unknown"},
			expectedType:  "unknown",
			expectedError: "Can't set the type unknown",
		},
		{
			description:  "parse string ident type",
			expr:         &ast.Ident{Name: "string"},
			expectedType: "string",
		},
		{
			description:  "parse bson ident type",
			expr:         &ast.Ident{Name: "bson"},
			expectedType: "string",
		},
		{
			description:  "parse integer ident type",
			expr:         &ast.Ident{Name: "int"},
			expectedType: "integer",
		},
		{
			description:  "parse int64 ident type",
			expr:         &ast.Ident{Name: "int64"},
			expectedType: "integer",
		},
		{
			description:  "parse int32 ident type",
			expr:         &ast.Ident{Name: "int32"},
			expectedType: "integer",
		},
		{
			description:    "parse time ident type",
			expr:           &ast.Ident{Name: "time"},
			expectedType:   "string",
			expectedFormat: "date-time",
		},
		{
			description:  "parse float64 ident type",
			expr:         &ast.Ident{Name: "float64"},
			expectedType: "number",
		},
		{
			description:  "parse bool ident type",
			expr:         &ast.Ident{Name: "bool"},
			expectedType: "boolean",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			tp, format, err := parseIdentProperty(tc.expr)
			if len(tc.expectedError) > 0 {
				if (err != nil) && (err.Error() != tc.expectedError) {
					t.Errorf("got error: %v, wantErr: %v", err, tc.expectedError)
				}
				if err == nil {
					t.Fatalf("expected error: %v . Got nothing", tc.expectedError)
				}
			}
			if (err != nil) && (len(tc.expectedError) == 0) {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.expectedType != tp {
				t.Errorf("got: %v, want: %v", tp, tc.expectedType)
			}
			if tc.expectedFormat != format {
				t.Errorf("got: %v, want: %v", format, tc.expectedFormat)
			}
		})
	}
}
