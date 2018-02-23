package docparser

import "testing"

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
