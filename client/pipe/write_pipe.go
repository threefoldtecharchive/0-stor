package pipe

import (
	"io"

	"github.com/zero-os/0-stor/client/config"
)

type WritePipe struct {
	w io.Writer
}

func NewWritePipe(conf config.Config) (*WritePipe, error) {
	w, err := conf.CreatePipeWriter(nil)
	if err != nil {
		return nil, err
	}
	return &WritePipe{
		w: w,
	}, nil
}
