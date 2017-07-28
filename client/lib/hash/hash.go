package hash

import (
	"fmt"
	"io"
)

// Hash Type
const (
	TypeBlake2 = "blake2_256"
	TypeSHA256 = "sha_256"
	TypeMD5    = "md5"
)

type hashEngine func([]byte) []byte

// Config defines hasher configuration
type Config struct {
	Type string
}

// Hasher is object that produces hash according to it's type
// given during it's creation
type Hasher struct {
	engine hashEngine
}

// Writer defines hash writer
type Writer struct {
	w      io.Writer
	hasher *Hasher
}

// NewWriter creates new hash writer
func NewWriter(w io.Writer, conf Config) (*Writer, error) {
	hasher, err := NewHasher(conf)
	if err != nil {
		return nil, err
	}
	return &Writer{
		w:      w,
		hasher: hasher,
	}, nil
}

// Write implements io.Writer interface
func (w Writer) Write(p []byte) (int, error) {
	hashed := w.hasher.Hash(p)
	return w.w.Write(hashed)
}

// NewHasher creates new hasher
func NewHasher(conf Config) (*Hasher, error) {
	var eng hashEngine

	switch conf.Type {
	case TypeBlake2:
		eng = blake2bHash
	case TypeSHA256:
		eng = sha256Hash
	case TypeMD5:
		eng = md5Hash
	default:
		return nil, fmt.Errorf("invalid hash type: %v", conf.Type)
	}
	return &Hasher{
		engine: eng,
	}, nil
}

// Hash produces hash of the given []byte
func (h Hasher) Hash(plain []byte) []byte {
	return h.engine(plain)
}
