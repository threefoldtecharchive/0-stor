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
		{"/ObjectManager/GetObject", stats.IncrRead, false},
		{"/ObjectManager/GetObjectStatus", stats.IncrRead, false},
		{"/ObjectManager/ListObjectKeys", stats.IncrRead, false},
		{"/ObjectManager/GetReferenceList", stats.IncrRead, false},
		{"/ObjectManager/GetReferenceCount", stats.IncrRead, false},
		{"/ObjectManager/SetObject", stats.IncrWrite, false},
		{"/ObjectManager/SetReferenceList", stats.IncrWrite, false},
		{"/ObjectManager/AppendToReferenceList", stats.IncrWrite, false},
		{"/ObjectManager/DeleteFromReferenceList", stats.IncrWrite, false},
		{"/ObjectManager/DeleteObject", stats.IncrWrite, false},
		{"/ObjectManager/DeleteReferenceList", stats.IncrWrite, false},
		{"/NamespaceManager/GetNamespace", stats.IncrRead, false},
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
