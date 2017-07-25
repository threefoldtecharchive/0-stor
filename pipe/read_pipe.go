package pipe

import (
	"github.com/zero-os/0-stor-lib/allreader"
)

type ReadPipe struct {
	readers []allreader.AllReader
}

func NewReadPipe(readers []allreader.AllReader) *ReadPipe {
	return &ReadPipe{
		readers: readers,
	}
}

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
