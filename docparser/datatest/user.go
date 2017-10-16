package cmd

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

// GetUser returns a user corresponding to specified id
// @openapi:path
// /pets:
//	get:
//		description: "Returns all pets from the system that the user has access to"
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
//				name: deviceId
//				schema:
//					type: integer
//				required: true
//				description: Numeric ID of the user to get
//		security:
//			- petstore_auth:
//				- write:pets
//				- read:pets
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

// Pet struct
// @openapi:schema
type Pet struct {
	ID              bson.ObjectId `json:"id"`
	String          string        `json:"string,omitempty" validate:"required"`
	Int             int           `json:"int,omitempty"`
	PointerOfString *string       `json:"pointerOfString"`
	SliceOfString   []string      `json:"sliceofString"`
	SliceOfInt      []int         `json:"sliceofInt"`
	Struct          Foo           `json:"struct"`
	SliceOfStruct   []Foo         `json:"sliceOfStruct"`
	PointerOfStruct *Foo          `json:"pointerOfStruct"`
	Time            time.Time     `json:"time"`
	PointerOfTime   *time.Time    `json:"pointerOfTime"`
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
