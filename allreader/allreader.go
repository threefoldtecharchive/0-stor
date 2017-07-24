package allreader

import (
	"io"
)

type AllReader interface {
	io.Reader
	ReadAll([]byte) ([]byte, error)
}
