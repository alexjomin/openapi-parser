# Parser OpenAPI [![Build Status](https://travis-ci.org/alexjomin/openapi-parser.svg?branch=master)](https://travis-ci.org/alexjomin/openapi-parser)

Parse openAPI from go comments in handlers and structs

## Install

+ `go get github.com/alexjomin/openapi-parser`

## Comments

### Path

The comments use the yaml syntax of the openapi specs. Just `@openapi:path` before the handler
**Be careful with the number of tabs**

```go
// GetUser returns a user corresponding to specified id
// @openapi:path
// /pets:
//	get:
//		description: "The description of the endpoint"
//		responses:
//			"200":
//				description: "The description of the response"
//				content:
//					application/json:
//						schema:
//							type: "array"
//							items:
//								$ref: "#/definitions/Foo"
//		parameters:
//			- in: path
//				name: deviceId
//				schema:
//					type: integer
//				required: true
//				description: Numeric ID of the user to get
func GetUser(w http.ResponseWriter, r *http.Request) {}
```

### Schema

The parser will parse the struct to create the shema, just add `@openapi:schema` before your struct

By default the name of the schema will be the name of the struct. You can overide it with `@openapi:schema:CustomName`. **Warning not all type are handled for now, work in progress.**

```go
// Foo struct
// @openapi:schema
type Pet Foo {
	String          string     `json:"string,omitempty"`
	Int             int        `json:"int,omitempty"`
	PointerOfString *string    `json:"pointerOfString"`
	SliceOfString   []string   `json:"sliceofString"`
	SliceOfInt      []int      `json:"sliceofInt"`
	Struct          Foo        `json:"struct"`
	PointerOfStruct *Foo       `json:"pointerOfStruct"`
	Time            time.Time  `json:"time"`
	PointerOfTime   *time.Time `json:"pointerOfTime"`
}
```

### Usage

```
Parse comments in code to generate an OpenAPI documentation

Usage:
  root [flags]

Flags:
  -h, --help            help for root
      --output string   The output file (default "openapi.yaml")
      --path string     The Folder to parse (default ".")
```

### Example

`parser-openapi`
`parser-openapi --path /my/path my-openapi.yaml`
