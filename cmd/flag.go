package cmd

import (
	"errors"
	"net"
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

// ListenAddress is a string representing a host and a port
// which can be used as a flag for a Cobra command.
type ListenAddress string

// String implements spf13/pflag.Value.String
func (b *ListenAddress) String() string {
	if len(*b) == 0 {
		return ":8080"
	}
	return string(*b)
}

// Set implements spf13/pflag.Value.Set
func (b *ListenAddress) Set(str string) error {
	host, _, err := net.SplitHostPort(str)
	if err != nil {
		return err
	}

	if ip := net.ParseIP(host); ip == nil {
		return errors.New("host not valid")
	}

	*b = ListenAddress(str)
	return nil
}

// Type implements spf13/pflag.Value.Type
func (b *ListenAddress) Type() string {
	return "listenAddress"
}
