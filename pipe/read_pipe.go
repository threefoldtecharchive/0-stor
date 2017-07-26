package pipe

import (
	"github.com/zero-os/0-stor-lib/allreader"
	"github.com/zero-os/0-stor-lib/config"
)

// ReadPipe represents pipe of readers
type ReadPipe struct {
	readers []allreader.AllReader
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

// ReadAll passes the data to the pipes
func (rp ReadPipe) ReadAll(data []byte) ([]byte, error) {
	var err error
	curData := data

	for _, rd := range rp.readers {
		curData, err = rd.ReadAll(curData)
		if err != nil {
			return nil, err
		}
	}
	return curData, nil
}
