package definition

import (
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
)

// NewCompiler returns a jsonschema.Compiler configured for fully support
// https://json-schema.org/draft/2019-09/schema
func NewCompiler() *jsonschema.Compiler {
	c := jsonschema.NewCompiler()
	c.DefaultDraft(jsonschema.Draft2019)
	c.AssertVocabs()
	c.AssertFormat()
	c.AssertContent()
	return c
}
