package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateName(t *testing.T) {
	type testcase struct {
		name  string
		valid bool
	}
	testcases := []testcase{
		testcase{name: "dasdf", valid: true},
		testcase{name: "d", valid: false},
		testcase{name: "dsf dfa-df", valid: true},
		testcase{name: "qwertyuiopasdfghjklmnbvczxqwerqwertyuiopasdfghjklmnbvczxqwer", valid: true},
		testcase{name: "qwertyuiopasdfghjklmnbvczxqwerdqwertyuiopasdfghjklmnbvczxqwerd", valid: false},
		testcase{name: "name with 'special' char", valid: true},
		testcase{name: "name with illeg@l char", valid: false},
		testcase{name: "дима", valid: true},
		testcase{name: "ратушный", valid: true},
	}
	for _, test := range testcases {
		assert.Equal(t, test.valid, ValidateName(test.name))
	}
}
