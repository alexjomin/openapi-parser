# Parser OpenAPI [![Build Status](https://travis-ci.org/alexjomin/openapi-parser.svg?branch=master)](https://travis-ci.org/alexjomin/openapi-parser)

Parse openAPI from go comments in handlers and structs

## Install

+ `go get github.com/alexjomin/openapi-parser`

## Comments

## Infos

The comments use the yaml syntax of the openapi specs. Just use the tag `@openapi:info` in your comments before the comments in which you want to set the openapi info fields.

```golang
// @openapi:info
//  version: 0.0.1
//  title: Some cool title
//  description: Awesome description
```

Note that only the first declaration of your info fields will be kept and inserted in the final yaml description file.

### Path

The comments use the yaml syntax of the openapi specs. Just `@openapi:path` before the handler
**Be careful with the number of tabs**

```golang
// GetUser returns a user corresponding to specified id
// @openapi:path
// /pets:
//    get:
//        description: "The description of the endpoint"
//        responses:
//            "200":
//                description: "The description of the response"
//                content:
//                    application/json:
//                        schema:
//                            type: "array"
//                            items:
//                                $ref: "#/components/schemas/Pet"
//        parameters:
//            - in: path
//                name: deviceId
//                schema:
//                    type: integer
//                required: true
//                description: Numeric ID of the user to get
func GetPets(w http.ResponseWriter, r *http.Request) {}
```

### Schema

The parser will parse the struct to create the shema, just add `@openapi:schema` before your struct

By default the name of the schema will be the name of the struct. You can overide it with `@openapi:schema:CustomName`. **Warning not all type are handled for now, work in progress.**

```golang
// Pet struct
// @openapi:schema
type Pet struct {
    String          string     `json:"string,omitempty"`
    Int             int        `json:"int,omitempty"`
    PointerOfString *string    `json:"pointerOfString"`
    SliceOfString   []string   `json:"sliceofString"`
    SliceOfInt      []int      `json:"sliceofInt"`
    Struct          Foo        `json:"struct"`
    SliceOfStruct   []Foo      `json:"sliceOfStruct"`
    PointerOfStruct *Foo       `json:"pointerOfStruct"`
    Time            time.Time  `json:"time"`
    PointerOfTime   *time.Time `json:"pointerOfTime"`
    EnumTest        string     `json:"enumTest" validate:"enum=UNKNOWN MALE FEMALE"`
}
```

for more, see `datatest/user`

### Usage

```text
Parse comments in code to generate an OpenAPI documentation

Usage:
  openapi-parser [flags]
  openapi-parser [command]

Available Commands:
  help        Help about any command
  merge       Merge multiple openapi specification into one

Flags:
  -h, --help            help for openapi-parser
      --output string   The output file (default "openapi.yaml")
      --path string     The Folder to parse (default ".")
```

### Example

`openapi-parser`

`openapi-parser --path /my/path --output my-openapi.yaml`
