package grpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zero-os/0-stor/server/stats"
)

func TestGetStatsfunc(t *testing.T) {
	assert := assert.New(t)
	cases := []struct {
		grpcMethod string
		statsFunc  labelStatsFunc
		err        bool
	}{
		{"/ObjectManager/Get", stats.IncrRead, false},
		{"/ObjectManager/List", stats.IncrRead, false},
		{"/ObjectManager/Exists", stats.IncrRead, false},
		{"/ObjectManager/Check", stats.IncrRead, false},
		{"/ObjectManager/Create", stats.IncrWrite, false},
		{"/ObjectManager/SetReferenceList", stats.IncrWrite, false},
		{"/ObjectManager/AppendReferenceList", stats.IncrWrite, false},
		{"/ObjectManager/RemoveReferenceList", stats.IncrWrite, false},
		{"/ObjectManager/Delete", stats.IncrWrite, false},
		{"/NamespaceManager/Get", stats.IncrRead, false},
		{"", nil, true},
		{"/ObjectManager/", nil, true},
		{"/NamespaceManager/", nil, true},
		{"/ObjectManager/Foo", nil, true},
		{"/NamespaceManager/Bar", nil, true},
	}

	for _, c := range cases {
		statsFunc, err := getStatsFunc(c.grpcMethod)
		if c.err {
			assert.Error(err)
		} else {
			assert.NoError(err)
			r1, w1 := stats.Rate(label)
			statsFunc(label)
			r2, w2 := stats.Rate(label)

			if r1 == r2 && w1 == w2 {
				assert.FailNow("Stats have not changed after statsFunc call")
			}
		}
	}
}
