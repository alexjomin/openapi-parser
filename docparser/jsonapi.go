package docparser

func addPrimaryFieldToJSONAPIData(data *schema, primaryType string) {
	data.Properties["id"] = schema{
		Type: "string",
	}
	data.Properties["type"] = schema{
		Type: "string",
		Enum: []string{primaryType},
	}
	data.Required = append(data.Required, "id", "type")
}
