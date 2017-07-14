package distribution

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDistributeRestore(t *testing.T) {
	const (
		k       = 4
		m       = 2
		dataLen = 4096
	)

	var writers []io.Writer
	var buffs []*bytes.Buffer
	for i := 0; i < k+m; i++ {
		buf := new(bytes.Buffer)
		buffs = append(buffs, buf)
		writers = append(writers, buf)
	}

	// distribute
	d, err := NewDistributor(k, m, writers)
	assert.Nil(t, err)

	data := make([]byte, dataLen)
	rand.Read(data)

	_, err = d.Write(data)
	assert.Nil(t, err)

	// restore
	var readers []io.Reader

	for i := 0; i < k+m; i++ {
		var reader io.Reader
		if i < m {
			// simulate losing pieces here
			// we can lost up to `m` pieces
			reader = bytes.NewReader([]byte("a"))
		} else {
			reader = bytes.NewReader(buffs[i].Bytes())
		}
		readers = append(readers, reader)
	}

	r, err := NewRestorer(k, m, readers)
	assert.Nil(t, err)

	decoded := make([]byte, len(data))

	n, err := r.Read(decoded)
	assert.Equal(t, len(data), n)
	assert.Nil(t, err)

	if bytes.Compare(decoded, data) != 0 {
		t.Fatal("restore failed")
	}
}

func TestDistributeDecode(t *testing.T) {
	const (
		k       = 4
		m       = 2
		dataLen = 4096
	)

	var writers []io.Writer
	var buffs []*bytes.Buffer
	for i := 0; i < k+m; i++ {
		buf := new(bytes.Buffer)
		buffs = append(buffs, buf)
		writers = append(writers, buf)
	}

	d, err := NewDistributor(k, m, writers)
	assert.Nil(t, err)

	data := make([]byte, dataLen)
	rand.Read(data)

	_, err = d.Write(data)
	assert.Nil(t, err)

	// decode
	var lost []int

	decodedChunks := make([][]byte, k+m)
	for i := 0; i < k+m; i++ {
		decodedChunks[i] = make([]byte, buffs[i].Len())

		if i < m {
			// simulate losing pieces here
			// we can lost up to `m` pieces
			lost = append(lost, i)
			continue
		}
		copy(decodedChunks[i], buffs[i].Bytes())
	}

	dec, err := NewDecoder(k, m)
	decoded, err := dec.Decode(decodedChunks, lost, len(data))
	assert.Nil(t, err)

	assert.Equal(t, data, decoded)

}
