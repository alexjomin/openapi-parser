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
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			parsedJSONTag, err := parseJSONTag(tc.field)
			if (err != nil) && (len(tc.expectedError) > 0) {
				if err.Error() != tc.expectedError {
					t.Errorf("got error: %v, wantErr: %v", err, tc.expectedError)
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
