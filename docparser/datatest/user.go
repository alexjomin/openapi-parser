package cmd

import (
	"encoding/json"
	"time"

	"gopkg.in/mgo.v2/bson"

	"github.com/alexjomin/openapi-parser/docparser/datatest/otherpackage"
)

// GetUser returns a user corresponding to specified id
// @openapi:path
// /pets:
//	get:
//		description: "Returns all pets from the system that the user has access to"
//		operationId: GetUser
//		tags:
//			- pet
//		responses:
//			"200":
//				description: "A list of pets."
//				content:
//					application/json:
//						schema:
//							type: "array"
//							items:
//								$ref: "#/components/schemas/Pet"
//			"302":
//				description: "Trip Signals Redirection"
//				headers:
//					Location:
//						description: The url to the signal API
//						schema:
//							type: string
//		parameters:
//			- in: path
//			  name: deviceId
//			  schema:
//			  	type: integer
//			  	enum: [3, 4]
//			  required: true
//			  description: Numeric ID of the user to get
//		security:
//			- petstore_auth:
//				- write:pets
//				- read:pets
//		servers:
//        - url: "https://{environment}.hello.com"
//          description: "what up"
//          variables:
//            environment:
//              default: api    # Production server
//              enum:
//                - api         # Production server
//                - api.dev     # Development server
//                - api.staging # Staging server
//		externalDocs:
//			description: External documentation
//			url: "https://{environment}-docs.hello.com"
func GetUser() {}

// PostFoo returns a user corresponding to specified id
// @openapi:path
// /pets:
//	post:
//		description: "Returns all pets from the system that the user has access to"
//		requestBody:
//			description: Pet to add to the store
//			required: true
//			content:
//				application/json:
//					schema:
//						$ref: "#/components/schemas/Pet"
//		responses:
//			"201":
//				description: "Post a new pet"
//				content:
//					application/json:
//						schema:
//							type: "array"
//							items:
//								$ref: "#/components/schemas/Pet"
func PostFoo() {}

// MapStringString type
// @openapi:schema
type MapStringString map[string]string

// @openapi:schema
type WeirdInt int

// Pet struct
// @openapi:schema
type Pet struct {
	ID                  bson.ObjectId             `json:"id"`
	String              string                    `json:"string,omitempty" validate:"required"`
	Int                 int                       `json:"int,omitempty"`
	PointerOfString     *string                   `json:"pointerOfString"`
	SliceOfString       []string                  `json:"sliceofString"`
	SliceOfInt          []int                     `json:"sliceofInt"`
	SliceOfSliceOfFloat [][]float64               `json:"sliceofSliceofFloat"`
	Struct              Foo                       `json:"struct"`
	SliceOfStruct       []Foo                     `json:"sliceOfStruct"`
	PointerOfStruct     *Foo                      `json:"pointerOfStruct"`
	Time                time.Time                 `json:"time"`
	PointerOfTime       *time.Time                `json:"pointerOfTime"`
	EnumTest            string                    `json:"enumTest" validate:"enum=UNKNOWN MALE FEMALE"`
	StrData             map[string]string         `json:"strData"`
	Children            map[string]Pet            `json:"children"`
	IntData             map[string]int            `json:"IntData"`
	ByteData            []byte                    `json:"ByteData"`
	JSONData            json.RawMessage           `json:"json_data"`
	CustomString        otherpackage.CustomString `json:"custom_string"`
	Test                Test                      `json:"test"`
}

// Dog struct
// @openapi:schema
type Dog struct {
	Pet
	otherpackage.Data

	Name string `json:"name"`
}

// Foo struct
// @openapi:schema
type Foo struct {
	String string `json:"string,omitempty"`
}

// Foo2 struct
// @openapi:schema:EditableFoo
type Foo2 struct {
	String string `json:"string,omitempty"`
}

// Signals struct
// @openapi:schema
type Signals []Foo

// Test struct
// @openapi:schema
type Test int

// @openapi:info
//  version: 0.0.1
//  title: Some cool title
//  description: Awesome description
