package config

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/zero-os/0-stor-lib/chunker"
	"github.com/zero-os/0-stor-lib/compress"
	"github.com/zero-os/0-stor-lib/distribution"
	"github.com/zero-os/0-stor-lib/encrypt"
	"github.com/zero-os/0-stor-lib/hash"
	"github.com/zero-os/0-stor-lib/replication"
)

func TestDecode(t *testing.T) {
	f, err := os.Open("./fixtures/simple.yaml")
	assert.Nil(t, err)

	conf, err := NewFromReader(f)
	assert.Nil(t, err)

	assert.Equal(t, conf.Namespace, "thedisk")
	assert.Equal(t, 6, len(conf.Pipes))

	p1 := conf.Pipes[0]
	compressConf, ok := p1.Config.(compress.Config)
	assert.True(t, ok)
	assert.Equal(t, 1, compressConf.Type)

	p2 := conf.Pipes[1]
	chunkerConf, ok := p2.Config.(chunker.Config)
	assert.True(t, ok)
	assert.Equal(t, 1, chunkerConf.ChunkSize)

	p3 := conf.Pipes[2]
	distConf, ok := p3.Config.(distribution.Config)
	assert.True(t, ok)
	assert.Equal(t, 1, distConf.K)

	p4 := conf.Pipes[3]
	encConf, ok := p4.Config.(encrypt.Config)
	assert.True(t, ok)
	assert.Equal(t, 1, encConf.Type)

	p5 := conf.Pipes[4]
	hConf, ok := p5.Config.(hash.Config)
	assert.True(t, ok)
	assert.Equal(t, 1, hConf.Type)

	p6 := conf.Pipes[5]
	rConf, ok := p6.Config.(replication.Config)
	assert.True(t, ok)
	assert.Equal(t, true, rConf.Async)

}

func TestWrite(t *testing.T) {
	// construct the config
	confPipe1 := compress.Config{
		Type: 1,
	}
	pipe1 := Pipe{
		Name:   "pipe1",
		Type:   "compress",
		Config: confPipe1,
	}
	conf := Config{
		Organization: "gig",
		Namespace:    "thedisk",
		IyoClientID:  "abc",
		IyoSecret:    "def",
		Shards:       []string{"http://127.0.0.1:12345", "http://127.0.0.1:12346"},
		Pipes:        []Pipe{pipe1},
	}

	buf := new(bytes.Buffer)

	// write it to the writer
	err := conf.Write(buf)
	assert.Nil(t, err)

	// compare with fixture
	fixture, err := ioutil.ReadFile("./fixtures/simple_write.yaml")
	assert.Nil(t, err)

	assert.Equal(t, string(fixture), string(buf.Bytes()))
}
