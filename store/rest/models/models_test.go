package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/store/db"
)

func TestModelsEncodeDecode(t *testing.T) {
	tt := []struct {
		model db.Model
		obj   interface{}
		name  string
	}{
		{
			model: &ACL{
				Acl: ACLEntry{},
				Id:  "foo",
			},
			name: "ACL",
			obj:  &ACL{},
		},
		{
			model: &Namespace{
				NamespaceCreate: NamespaceCreate{
					Label: "foo",
					Acl:   []ACL{},
				},
				SpaceAvailable: 100,
				SpaceUsed:      50,
			},
			name: "Namespace",
			obj:  &Namespace{},
		},
		{
			model: &File{
				Namespace: "foo",
				Id:        "ID",
				Reference: byte(1),
				Payload:   "hello world",
				Tags: []Tag{
					{
						Key:   "hi",
						Value: "there",
					},
				},
			},
			name: "File",
			obj:  &File{},
		},
		{
			model: &StoreStat{
				SizeAvailable: 100,
				SizeUsed:      50,
			},
			name: "StoreStat",
			obj:  &StoreStat{},
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			b, err := test.model.Encode()
			require.NoError(t, err)
			model := test.obj.(db.Model)
			err = model.Decode(b)
			require.NoError(t, err)
			assert.Equal(t, test.model, model)
		})
	}
}
