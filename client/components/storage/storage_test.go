package storage

import (
	"crypto/rand"
	"fmt"
	"math"
	mathRand "math/rand"
	"testing"

	"github.com/zero-os/0-stor/client/datastor"

	"github.com/stretchr/testify/require"
)

func testStorageReadCheckWrite(t *testing.T, storage ObjectStorage) {
	require.NotNil(t, storage)

	t.Run("fixed test cases", func(t *testing.T) {
		require := require.New(t)

		objects := []datastor.Object{
			datastor.Object{
				Key:  []byte("a"),
				Data: []byte("b"),
			},
			datastor.Object{
				Key:  []byte("foo"),
				Data: []byte("bar"),
			},
			datastor.Object{
				Key:  []byte("大家好"),
				Data: []byte("大家好"),
			},
			datastor.Object{
				Key:           []byte("this-is-my-key"),
				Data:          []byte("Hello, World!"),
				ReferenceList: []string{"user1", "user2"},
			},
			datastor.Object{
				Key:           []byte("this-is-my-key"),
				Data:          []byte("Hello, World!"),
				ReferenceList: []string{"user1", "user2"},
			},
		}
		for _, inputObject := range objects {
			// write object & validate
			cfg, err := storage.Write(inputObject)
			require.NoError(err)
			require.Equal(inputObject.Key, cfg.Key)
			require.Equal(len(inputObject.Data), cfg.DataSize)

			// validate that all shards contain valid data
			status, err := storage.Check(cfg, false)
			require.NoError(err)
			require.Equal(ObjectCheckStatusOptimal, status)

			// read object & validate
			outputObject, err := storage.Read(cfg)
			require.NoError(err)
			require.Equal(inputObject, outputObject)
		}
	})

	t.Run("random test cases", func(t *testing.T) {
		require := require.New(t)

		for i := 0; i < 256; i++ {
			key := []byte(fmt.Sprintf("key#%d", i+1))
			data := make([]byte, mathRand.Int31n(128)+1)
			rand.Read(data)

			refList := make([]string, mathRand.Int31n(8)+1)
			for i := range refList {
				id := make([]byte, mathRand.Int31n(32)+1)
				rand.Read(id)
				refList[i] = string(id)
			}

			inputObject := datastor.Object{
				Key:           key,
				Data:          data,
				ReferenceList: refList,
			}

			// write object & validate
			cfg, err := storage.Write(inputObject)
			require.NoError(err)
			require.Equal(inputObject.Key, cfg.Key)
			require.Equal(len(data), cfg.DataSize)

			// validate that all shards contain valid data
			status, err := storage.Check(cfg, false)
			require.NoError(err)
			require.Equal(ObjectCheckStatusOptimal, status)

			// read object & validate
			outputObject, err := storage.Read(cfg)
			require.NoError(err)
			require.Equal(inputObject, outputObject)
		}
	})
}

func TestObjectCheckStatusString(t *testing.T) {
	require := require.New(t)

	// valid enum values
	require.Equal("invalid", ObjectCheckStatusInvalid.String())
	require.Equal("valid", ObjectCheckStatusValid.String())
	require.Equal("optimal", ObjectCheckStatusOptimal.String())

	// invalid enum value
	require.Empty(ObjectCheckStatus(math.MaxUint8).String())
}
