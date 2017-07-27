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
		dataLen = 4096
	)
	conf := Config{
		K: 4,
		M: 2,
	}

	var writers []io.Writer
	var buffs []*bytes.Buffer
	for i := 0; i < conf.NumPieces(); i++ {
		buf := new(bytes.Buffer)
		buffs = append(buffs, buf)
		writers = append(writers, buf)
	}

	// distribute
	d, err := NewDistributor(writers, conf)
	assert.Nil(t, err)

	data := make([]byte, dataLen)
	rand.Read(data)

	_, err = d.Write(data)
	assert.Nil(t, err)

	// restore
	var readers []io.Reader

	for i := 0; i < conf.NumPieces(); i++ {
		var reader io.Reader
		if i < conf.M {
			// simulate losing pieces here
			// we can lost up to `m` pieces
			reader = bytes.NewReader(nil)
		} else {
			reader = bytes.NewReader(buffs[i].Bytes())
		}
		readers = append(readers, reader)
	}

	r, err := NewRestorer(readers, conf)
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
		dataLen = 4096
	)
	conf := Config{
		K: 4,
		M: 2,
	}

	var writers []io.Writer
	var buffs []*bytes.Buffer
	for i := 0; i < conf.NumPieces(); i++ {
		buf := new(bytes.Buffer)
		buffs = append(buffs, buf)
		writers = append(writers, buf)
	}

	d, err := NewDistributor(writers, conf)
	assert.Nil(t, err)

	data := make([]byte, dataLen)
	rand.Read(data)

	_, err = d.Write(data)
	assert.Nil(t, err)

	// decode

	decodedChunks := make([][]byte, conf.NumPieces())
	for i := 0; i < conf.NumPieces(); i++ {

		if i < conf.M {
			// simulate losing pieces here
			// we can lost up to `m` pieces
			continue
		}
		decodedChunks[i] = make([]byte, buffs[i].Len())
		copy(decodedChunks[i], buffs[i].Bytes())
	}

	dec, err := NewDecoder(conf.K, conf.M)
	decoded, err := dec.Decode(decodedChunks, len(data))
	assert.Nil(t, err)

	if !assert.Equal(t, data, decoded) {
		return
	}

}
