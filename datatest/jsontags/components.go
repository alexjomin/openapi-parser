package jsontags

// @openapi:info
//  version: 1.2.3
//  title: jsontags
//  description: Datatest for json tags

// Bar struct
// @openapi:schema
type Bar struct {
	RequiredField                string `json:"requiredField" validate:"required"`
	RequiredButNotValidatedField string `json:"requiredButNotValidatedField" validate:"-,required"`
	NullableField                string `json:"nullableField,omitempty"`
}

// Baz struct
// @openapi:schema
type Baz struct {
	StringAsString  string  `json:"stringAsString,string"`
	IntAsString     int     `json:"intAsString,string"`
	Float64AsString float64 `json:"float64AsString,string"`
	BooleanAsString bool    `json:"booleanAsString,string"`
}
