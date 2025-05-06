package definition

import (
	"regexp"
)

type Regexp struct {
	regexp.Regexp
}

func (r Regexp) MarshalBinary() ([]byte, error) {
	return []byte(r.String()), nil
}

// UnmarshalBinary modifies the receiver so it must take a pointer receiver.
func (r *Regexp) UnmarshalBinary(data []byte) error {
	return r.UnmarshalText(data)
}
