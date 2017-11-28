package cmd

import (
	"errors"
	"strings"
)

// Strings is a slice of strings which can be used as
// a flag for a Cobra command.
type Strings []string

// String implements spf13/pflag.Value.String
func (s Strings) String() string {
	if len(s) == 0 {
		return ""
	}
	return strings.Join([]string(s), ",")
}

// Strings returns this Slice as a []string value.
func (s Strings) Strings() []string {
	return s
}

// Set implements spf13/pflag.Value.Set
func (s *Strings) Set(str string) error {
	if len(str) == 0 {
		return errors.New("no strings given")
	}
	*s = strings.Split(str, ",")
	return nil
}

// Type implements spf13/pflag.Value.Type
func (s Strings) Type() string {
	return "strings"
}
