package docparser

import (
	"go/ast"
	"reflect"
	"testing"
)

type parseJSONTagTestCase struct {
	description     string
	field           *ast.Field
	expectedJSONTag jsonTagInfo
	expectedError   string
}

type parseIdentPropertyTestCase struct {
	description    string
	expr           *ast.Ident
	expectedType   string
	expectedError  string
	expectedFormat string
}

func TestParseJSONTag(t *testing.T) {
	testCases := []parseJSONTagTestCase{
		{
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
				ignore:   true,
				required: false,
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
		{
			description: "Should parse json tag value and validate",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`json:\"jsontagname\" validate:\"required\"`",
				},
			},
			expectedJSONTag: jsonTagInfo{
				required: true,
				name:     "jsontagname",
			},
		},
		{
			description: "Should parse json tag value and validate with enum",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`json:\"jsontagname\" validate:\"required,enum=a b\"`",
				},
			},
			expectedJSONTag: jsonTagInfo{
				required: true,
				name:     "jsontagname",
				enum:     []string{"a", "b"},
			},
		},
		{
			description: "Should use Tag name rather than ident name",
			field: &ast.Field{
				Tag: &ast.BasicLit{
					ValuePos: 0,
					Kind:     0,
					Value:    "`json:\"jsontagname\" validate:\"required,enum=a b\"`",
				},
				Names: []*ast.Ident{
					&ast.Ident{
						Name: "testName",
					},
				},
			},
			expectedJSONTag: jsonTagInfo{
				required: true,
				name:     "jsontagname",
				enum:     []string{"a", "b"},
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
			if tc.expectedError != "" {
				assert.Equal(t, tc.expectedError, err.Error())
				return
			}
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedType, tp)
			assert.Equal(t, tc.expectedFormat, format)
		})
	}
}
