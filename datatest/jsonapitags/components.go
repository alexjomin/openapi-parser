package jsonapitags

// @openapi:info
//  version: 1.2.3
//  title: jsonapitags
//  description: Datatest for jsonapi tags

// Foo struct
// @openapi:schema
type Foo struct {
	ID                     string `jsonapi:"primary,foos"`
	AttributeField         string `jsonapi:"attr,field2"`
	NullableAttributeField string `jsonapi:"attr,field3,omitempty"`
	Reference              Bar    `jsonapi:"relation,ref1"`
	ReferencePtr           *Baz   `jsonapi:"relation,ref2"`
}

// Bar struct
// @openapi:schema
type Bar struct {
	ID                string `jsonapi:"primary,bars"`
	AttributeField    string `jsonapi:"attr,field4"`
	ReferenceArray    []Baz  `jsonapi:"relation,ref3"`
	ReferenceArrayPtr []*Bar `jsonapi:"relation,ref4"`
}

// Baz struct
// @openapi:schema
type Baz struct {
	ID             string `jsonapi:"primary,bazs"`
	AttributeField string `jsonapi:"attr,field5"`
	Reference      Qux    `jsonapi:"relation,ref6"`
}

// Qux struct
// @openapi:schema
type Qux struct {
	ID             string `jsonapi:"primary,quxs"`
	AttributeField string `jsonapi:"attr,field7"`
}

// ErrorsPayload struct
// @openapi:schema
type ErrorsPayload struct {
	Errors []*ErrorObject `json:"errors"`
}

// ErrorObject struct
// @openapi:schema
type ErrorObject struct {
	Title  string `json:"title"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
	Status int    `json:"status"`
}
