package cmd

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListenAddressFlag(t *testing.T) {
	tt := []struct {
		input string
		err   error
		value string
	}{
		{
			// default value
			"",
			nil,
			":8080",
		},
		{
			"8080",
			errors.New("address 8080: missing port in address"),
			"",
		},
		{
			":8080",
			nil,
			":8080",
		},
		{
			"127.0.0.1:8080",
			nil,
			"127.0.0.1:8080",
		},
		{
			"badhost:8080",
			errors.New("host not valid"),
			"",
		},
	}

	for _, tc := range tt {
		t.Run(tc.input, func(t *testing.T) {
			require := require.New(t)

			var (
				l   ListenAddress
				err error
			)

			if tc.input != "" {
				err = l.Set(tc.input)
			}
			if tc.err != nil {
				require.Equal(tc.err.Error(), err.Error())
			} else {
				require.Equal(tc.value, l.String())
			}
		})
	}
}
