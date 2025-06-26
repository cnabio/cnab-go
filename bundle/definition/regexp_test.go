package definition_test

import (
	"bytes"
	"regexp"
	"testing"

	"encoding/gob"

	"github.com/cnabio/cnab-go/bundle/definition"
)

func TestRegexp_Encode_Decode(t *testing.T) {
	type fields struct {
		Regexp regexp.Regexp
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "1",
			fields: fields{
				Regexp: *regexp.MustCompile(`/^(\\([0-9]{3}\\))?[0-9]{3}-[0-9]{4}$/u`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			r := definition.Regexp{
				Regexp: tt.fields.Regexp,
			}
			err := enc.Encode(r)
			if err != nil {
				t.Error(err)
			}

			dec := gob.NewDecoder(&buf)
			v := definition.Regexp{}
			err = dec.Decode(&v)
			if err != nil {
				t.Error(err)
			}

			if v.String() != r.String() {
				t.Errorf("Regexp.MarshalBinary() = %v, want %v", v.String(), r.String())
			}
		})
	}
}
