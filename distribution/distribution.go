package distribution

import (
	"fmt"
	"io"
	"io/ioutil"
)

// Distributor distribute the data to the given outputs
type Distributor struct {
	enc     *Encoder
	writers []io.Writer
}

// Config defines distribution's configuration
type Config struct {
	K int `yaml:"k"`
	M int `yaml:"m"`
}

// NumPieces returns total number of pieces given the configuration
func (c Config) NumPieces() int {
	return c.K + c.M
}

// NewDistributor creates new distribution
func NewDistributor(writers []io.Writer, conf Config) (*Distributor, error) {
	if len(writers) != conf.K+conf.M {
		return nil, fmt.Errorf("invalid number of writers: %v expected: %v", len(writers), conf.K+conf.M)
	}

	enc, err := NewEncoder(conf.K, conf.M)
	if err != nil {
		return nil, err
	}

	return &Distributor{
		enc:     enc,
		writers: writers,
	}, nil
}

// Write writes blocks to the given output writers
func (d Distributor) Write(data []byte) (int, error) {
	var written int

	encoded, err := d.enc.Encode(data)
	if err != nil {
		return written, err
	}

	for i, data := range encoded {
		n, err := d.writers[i].Write(data)
		written += n
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

// Restorer restore the distributed data from the given readers
type Restorer struct {
	dec     *Decoder
	readers []io.Reader
}

// NewRestorer creates restorer from the given reader.
func NewRestorer(readers []io.Reader, conf Config) (*Restorer, error) {
	if len(readers) != conf.K+conf.M {
		return nil, fmt.Errorf("invalid number of readers: %v expected: %v", len(readers), conf.K+conf.M)
	}

	dec, err := NewDecoder(conf.K, conf.M)
	if err != nil {
		return nil, err
	}
	return &Restorer{
		dec:     dec,
		readers: readers,
	}, nil
}

// Read restores the data from the underlying reader.
// length of the decoded argument must be the same as lenght of
// the original data
func (r *Restorer) Read(decoded []byte) (int, error) {
	var lost []int

	origLen := len(decoded)
	chunks := make([][]byte, r.dec.k+r.dec.m)
	chunkLen := getPaddedLen(origLen, r.dec.k) / r.dec.k

	// read all chunks from the underlying reader
	for i, reader := range r.readers {
		data, err := ioutil.ReadAll(reader)
		if err != nil || len(data) != chunkLen {
			// error and invalid lenght are marked as data lost!
			lost = append(lost, i)
			chunks[i] = make([]byte, chunkLen)
		} else {
			chunks[i] = data
		}
	}

	// decode
	res, err := r.dec.Decode(chunks, lost, origLen)
	if err != nil || len(res) != origLen {
		return len(res), err
	}

	copy(decoded, res)
	return len(res), nil
}
