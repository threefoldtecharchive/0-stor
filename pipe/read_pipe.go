package pipe

import (
	"github.com/zero-os/0-stor-lib/config"
	"github.com/zero-os/0-stor-lib/fullreadwrite"
)

// ReadPipe represents pipe of readers
type ReadPipe struct {
	readers []fullreadwrite.Reader
}

// NewReadPipe create ReadPipe from config
func NewReadPipe(conf config.Config) (*ReadPipe, error) {
	ars, err := conf.CreateAllReaders()
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
		curData, err = rd.ReadFull(curData)
		if err != nil {
			return nil, err
		}
	}
	return curData, nil
}
