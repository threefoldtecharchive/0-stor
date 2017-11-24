package jwt

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethodString(t *testing.T) {
	cases := []struct {
		Value    Method
		Expected string
	}{
		{MethodRead, "read"},
		{MethodWrite, "write"},
		{MethodDelete, "delete"},
		{MethodAdmin, "admin"},
		{Expected: ""},
		{42, ""},
	}

	for _, c := range cases {
		assert.Equalf(t, c.Expected, c.Value.String(), "Method: %d", c.Value)
		assert.Equalf(t, c.Expected, fmt.Sprint(c.Value), "Method: %d", c.Value)
		assert.Equalf(t, c.Expected, fmt.Sprintf("%s", c.Value), "Method: %d", c.Value)
	}
}
