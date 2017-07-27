package distribution

import (
	"fmt"

	"github.com/templexxx/reedsolomon"
)

const (
	padFactor = reedsolomon.LoopSizeAVX2
)

// Encoder encode the data to be distributed
// Use this object instead of the distribution if you
// don't have Writers for your output
type Encoder struct {
	k   int
	m   int
	enc reedsolomon.EncodeReconster // encoder  & decoder
}

// NewEncoder creates new encoder
func NewEncoder(k, m int) (*Encoder, error) {
	ed, err := reedsolomon.New(k, m)
	if err != nil {
		return nil, err
	}

	return &Encoder{
		k:   k,
		m:   m,
		enc: ed,
	}, nil
}

// Encode encodes the data using erasure code
func (enc *Encoder) Encode(data []byte) ([][]byte, error) {
	datas := enc.splitData(data)

	parities := reedsolomon.NewMatrix(enc.m, len(datas[0]))
	err := enc.enc.Encode(datas, parities)
	return append(datas, parities...), err
}

func (enc *Encoder) splitData(data []byte) [][]byte {
	data = enc.padIfNeeded(data)
	chunkSize := len(data) / enc.k
	chunks := make([][]byte, enc.k)

	for i := 0; i < enc.k; i++ {
		chunks[i] = data[i*chunkSize : (i+1)*chunkSize]
	}
	return chunks
}

// add padding if needed
func (enc *Encoder) padIfNeeded(data []byte) []byte {
	padLen := getPadLen(len(data), enc.k)
	if padLen == 0 {
		return data
	}

	pad := make([]byte, padLen)
	return append(data, pad...)
}

func getPadLen(dataLen, k int) int {
	maxPadLen := k * padFactor
	mod := dataLen % maxPadLen
	if mod == 0 {
		return 0
	}
	return maxPadLen - mod
}

func getPaddedLen(dataLen, k int) int {
	return dataLen + getPadLen(dataLen, k)
}

// Decoder decodes the encoded data
type Decoder struct {
	k   int
	m   int
	dec reedsolomon.EncodeReconster // encoder  & decoder
}

// NewDecoder creates new decoder
func NewDecoder(k, m int) (*Decoder, error) {
	ed, err := reedsolomon.New(k, m)
	if err != nil {
		return nil, err
	}

	return &Decoder{
		k:   k,
		m:   m,
		dec: ed,
	}, nil
}

// Decode decodes the data using erasure code.
// Lost is array of lost pieces index.
// origLen is the original data length.
func (d *Decoder) Decode(chunks [][]byte, lost []int, origLen int) ([]byte, error) {
	if len(lost) > d.m {
		return nil, fmt.Errorf("too many lost chunks: %v, max: %v", len(lost), d.m)
	}

	// get pieces we have from pieces we lost
	have := func() []int {
		var have []int

		// build map of lost pieces for easier checking
		lostMap := make(map[int]struct{})
		for _, v := range lost {
			lostMap[v] = struct{}{}
		}
		for i := 0; i < len(chunks); i++ {
			if _, ok := lostMap[i]; !ok {
				have = append(have, i)
			}
		}
		return have
	}()

	// decode
	if err := d.dec.Reconst(chunks, have, lost); err != nil {
		return nil, err
	}

	// merge the chunks
	decoded := make([]byte, 0, origLen)
	for i := 0; i < d.k; i++ {
		decoded = append(decoded, chunks[i]...)
	}
	return decoded[:origLen], nil
}
