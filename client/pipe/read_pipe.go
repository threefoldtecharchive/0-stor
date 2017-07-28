package pipe

import (
	"github.com/zero-os/0-stor/client/config"
	"github.com/zero-os/0-stor/client/lib/block"
)

// ReadPipe represents pipe of readers
type ReadPipe struct {
	readers []block.Reader
}

func createAllBlockReaders(conf *config.Config) ([]block.Reader, error) {
	var readers []block.Reader
	for _, pipe := range conf.Pipes {
		ar, err := pipe.CreateBlockReader(conf.Shards, conf.Organization, conf.Namespace)
		if err != nil {
			return nil, err
		}
		readers = append([]block.Reader{ar}, readers...)
	}
	return readers, nil
}

// NewReadPipe create ReadPipe from config
func NewReadPipe(conf *config.Config) (*ReadPipe, error) {
	ars, err := createAllBlockReaders(conf)
	if err != nil {
		return nil, err
	}
	return &ReadPipe{
		readers: ars,
	}, nil
}

// ReadFull passes the data to the pipes
func (rp ReadPipe) ReadFull(data []byte) ([]byte, error) {
	var err error
	curData := data

	for _, rd := range rp.readers {
		curData, err = rd.ReadBlock(curData)
		if err != nil {
			return nil, err
		}
	}
	return curData, nil
}
