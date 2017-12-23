package badger

import (
	"context"
	"fmt"
	mathRand "math/rand"
	"strings"
	"testing"

	"golang.org/x/sync/errgroup"

	"github.com/stretchr/testify/require"
)

func TestAsyncCache(t *testing.T) {
	// generate keys
	const (
		keyCount   = 512
		keyMaxSize = 32
		keyMinSize = 8
	)
	keys := make([][]byte, keyCount)
	const chars = "abcdefghijklmnopqrtsuvwxyzABCDEFGHIJKLMNOPQRSTUVXZYZ-@_+"
	for i := range keys {
		id := make([]byte, mathRand.Int31n(keyMaxSize-keyMinSize)+keyMinSize)
		for u := range id {
			id[u] = chars[mathRand.Int31n(int32(len(chars)))]
		}
		keys[i] = []byte(fmt.Sprintf("d:%d:%s", i, id))
	}

	// create test db
	db, cleanup := makeTestBadgerDB(t)
	defer cleanup()

	cache := newSequenceCache(db.db)
	require.NotNil(t, cache)
	defer cache.Purge()

	// increment all counters

	const (
		incrementCount = 256
		workerCount    = keyCount * 2
		countPerWorker = incrementCount / 2
	)
	type output struct {
		ScopeKey, Key string
	}
	outputCh := make(chan output, workerCount*2)

	group, ctx := errgroup.WithContext(context.Background())
	// create multiple workers per key
	for i := 0; i < workerCount; i++ {
		index := i % keyCount
		group.Go(func() error {
			for i := 0; i < countPerWorker; i++ {
				key, err := cache.IncrementKey(keys[index])
				if err != nil {
					return err
				}
				select {
				case outputCh <- output{string(keys[index]), string(key)}:
				case <-ctx.Done():
					return nil
				}
			}
			return nil
		})
	}
	go func() {
		err := group.Wait()
		close(outputCh)
		require.NoError(t, err)
	}()

	allOutput := make(map[string]map[string]struct{}, keyCount)
	for _, key := range keys {
		allOutput[string(key)] = make(map[string]struct{})
	}
	for output := range outputCh {
		require.True(t, strings.HasPrefix(output.Key, output.ScopeKey))

		m, ok := allOutput[output.ScopeKey]
		require.True(t, ok)

		_, ok = m[output.Key]
		require.False(t, ok)
		m[output.Key] = struct{}{}
	}
	for _, m := range allOutput {
		require.Len(t, m, incrementCount)
	}
}
