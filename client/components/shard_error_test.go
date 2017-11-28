package components

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShardError(t *testing.T) {
	require := require.New(t)

	var shardErr ShardError

	require.True(shardErr.Nil(), "must return true if no errors")

	require.Equal(shardErr.Num(), 0, "incorrect number of errors")
	require.True(shardErr.Error() == "", "empty error message expected")

	// add an error for one shard
	shardErr.Add([]string{"22379"}, "", nil, 0)

	require.False(shardErr.Nil(), "must return false if any errors")

	require.Equal(shardErr.Num(), 1, "incorrect number of errors")

	// add an error for another shard
	shardErr.Add([]string{"32379", "2379"}, "", nil, 0)

	require.Equal(shardErr.Num(), 2, "incorrect number of errors")
	require.False(shardErr.Error() == "", "non-empty error message expected")
}

func TestThreadSafe(t *testing.T) {
	// define number of threads
	const n = 1000
	var shardErr ShardError
	var wg sync.WaitGroup

	// create n threads to add errors to shardErr
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			shardErr.Add([]string{"22379"}, "", nil, 0)
		}()
	}
	wg.Wait()

	require.Equal(t, shardErr.Num(), n, "incorrect number of errors, race condition possible")
}
