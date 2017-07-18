package encrypt

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
)

// Encryption type
const (
	_ = iota
	TypeAESGCM
)

var (
	errInvalidPrivKeyLen = errors.New("invalid private key length")
	errInvalidNonceLen   = errors.New("invalid nonce length")
)

// Config defines EncrypterDecrypter config
type Config struct {
	Type    int    `yaml:"type"`
	PrivKey []byte `yaml:"privKey"`
	Nonce   []byte `yaml:"nonce"`
}

// EncrypterDecrypter is interaface for encrypter and decrypter
type EncrypterDecrypter interface {
	Encrypt(plain []byte) []byte
	Decrypt(cipher []byte) (plain []byte, err error)
}

// NewEncrypterDecrypter creates new EncrypterDecrypter
func NewEncrypterDecrypter(conf Config) (EncrypterDecrypter, error) {
	switch conf.Type {
	case TypeAESGCM:
		return newAESGCM(conf.PrivKey, conf.Nonce)
	default:
		return nil, fmt.Errorf("invalid type: %v", conf.Type)
	}
}

// Writer defines encryption writer
type Writer struct {
	ed EncrypterDecrypter
	w  io.Writer
}

// NewWriter creates new encryption writer
func NewWriter(w io.Writer, conf Config) (*Writer, error) {
	ed, err := NewEncrypterDecrypter(conf)
	if err != nil {
		return nil, err
	}
	return &Writer{
		w:  w,
		ed: ed,
	}, nil
}

// Writers implements io.Writer interface
func (w *Writer) Write(plain []byte) (int, error) {
	encrypted := w.ed.Encrypt(plain)
	return w.w.Write(encrypted)
}

// Reader defines encryption reader.
// It use ioutil.ReadAll so it won't save your memory usage
type Reader struct {
	ed EncrypterDecrypter
	rd io.Reader
}

// NewReader creates new encryption reader
func NewReader(rd io.Reader, conf Config) (*Reader, error) {
	ed, err := NewEncrypterDecrypter(conf)
	if err != nil {
		return nil, err
	}
	return &Reader{
		rd: rd,
		ed: ed,
	}, nil
}

// Read implements io.Reader.
// the given `plain []byte` must have enough size to hold the data
func (r *Reader) Read(plain []byte) (int, error) {
	encrypted, err := ioutil.ReadAll(r.rd)
	if err != nil {
		return 0, err
	}

	decrypted, err := r.ed.Decrypt(encrypted)
	if err != nil {
		return 0, err
	}

	copy(plain, decrypted)
	return len(decrypted), nil
}
