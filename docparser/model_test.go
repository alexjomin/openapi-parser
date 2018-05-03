package docparser

import (
	"go/ast"
	"testing"

	"github.com/stretchr/testify/assert"
)

type parseInfosTestCase struct {
	description         string
	gofiles             []*ast.File
	expectedVersion     string
	expectedTitle       string
	expectedDescription string
}

func Test_validatePath(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"Vendor path",
			args{
				path: "/foo/bar/test/vendor/test.go",
			},
			false,
		},
		{
			"No go file path",
			args{
				path: "/foo/bar/test/test.py",
			},
			false,
		},
		{
			"Dot File",
			args{
				path: ".DS_STORE",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validatePath(tt.args.path); got != tt.want {
				t.Errorf("validatePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseInfos(t *testing.T) {
	commentsWithoutInfoTag := &ast.CommentGroup{
		List: []*ast.Comment{
			&ast.Comment{Text: "// Some useless comment"},
			&ast.Comment{Text: "// Another useless comment"},
		},
	}

	commentsWithInfoTagOnly := &ast.CommentGroup{
		List: []*ast.Comment{
			&ast.Comment{Text: "// @openapi:info"},
			&ast.Comment{Text: "// Another useless comment"},
		},
	}

	commentsWithInfoTagAndData := &ast.CommentGroup{
		List: []*ast.Comment{
			&ast.Comment{Text: "// @openapi:info"},
			&ast.Comment{Text: `// version: "1.0.1"`},
			&ast.Comment{Text: `// title: "some cool title"`},
			&ast.Comment{Text: `// description: "some cool description tho"`},
		},
	}

	commentsWithInfoDataOnly := &ast.CommentGroup{
		List: []*ast.Comment{
			&ast.Comment{Text: `// version: "1.0.1"`},
			&ast.Comment{Text: `// title: "some cool title"`},
			&ast.Comment{Text: `// description: "some cool description tho"`},
		},
	}

	commentsWithInfoTagAndDataDuplicated := &ast.CommentGroup{
		List: []*ast.Comment{
			&ast.Comment{Text: "// @openapi:info"},
			&ast.Comment{Text: `// version: "1.4"`},
			&ast.Comment{Text: `// title: "another cool title"`},
			&ast.Comment{Text: `// description: "another cool description tho"`},
		},
	}

	testCases := []parseInfosTestCase{
		{
			description: "Parse comments not containing info tags shouldn't change Info data",
			gofiles: []*ast.File{
				&ast.File{Comments: []*ast.CommentGroup{commentsWithoutInfoTag}},
			},
			expectedVersion:     "",
			expectedTitle:       "",
			expectedDescription: "",
		},
		{
			description: "Parse comments containing only info tag shouldn't change Info data",
			gofiles: []*ast.File{
				&ast.File{Comments: []*ast.CommentGroup{commentsWithInfoTagOnly}},
			},
			expectedVersion:     "",
			expectedTitle:       "",
			expectedDescription: "",
		},
		{
			description: "Parse comments containing info tag and infos data should set Info data",
			gofiles: []*ast.File{
				&ast.File{Comments: []*ast.CommentGroup{commentsWithInfoTagAndData}},
			},
			expectedVersion:     "1.0.1",
			expectedTitle:       "some cool title",
			expectedDescription: "some cool description tho",
		},
		{
			description: "Parse comments containing only infos data shouldn't change Info data",
			gofiles: []*ast.File{
				&ast.File{Comments: []*ast.CommentGroup{commentsWithInfoDataOnly}},
			},
			expectedVersion:     "",
			expectedTitle:       "",
			expectedDescription: "",
		},
		{
			description: "Parse comments containing info tag and duplicated infos data should set Info data with the first values parsed",
			gofiles: []*ast.File{
				&ast.File{Comments: []*ast.CommentGroup{commentsWithInfoTagAndData}},
				&ast.File{Comments: []*ast.CommentGroup{commentsWithInfoTagAndDataDuplicated}},
			},
			expectedVersion:     "1.0.1",
			expectedTitle:       "some cool title",
			expectedDescription: "some cool description tho",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			spec := NewOpenAPI(false)
			for _, gofile := range tc.gofiles {
				spec.parseInfos(gofile)
			}
			assert.Equal(t, tc.expectedVersion, spec.Info.Version)
			assert.Equal(t, tc.expectedTitle, spec.Info.Title)
			assert.Equal(t, tc.expectedDescription, spec.Info.Description)
		})
	}
}
