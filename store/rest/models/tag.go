package models

import (
	validator "gopkg.in/validator.v2"
	"encoding/binary"
)


// TagHeader
type TagHeader struct{
	KeyLen int64
	ValueLen int64
}

func (h TagHeader) Encode() ([]byte, error){
	r := make([]byte, 16)
	binary.LittleEndian.PutUint64(r[0:8], uint64(h.KeyLen))
	binary.LittleEndian.PutUint64(r[8:16], uint64(h.ValueLen))
	return r, nil
}

func (h *TagHeader) Decode(data []byte) error{
	h.KeyLen = int64(binary.LittleEndian.Uint64(data[0:8]))
	h.ValueLen = int64(binary.LittleEndian.Uint64(data[8:16]))
	return nil
}

// Tag
type Tag struct {
	Key   string `json:"key" validate:"regexp=^\w+$,nonzero"`
	Value string `json:"value" validate:"nonzero"`
}


func (t Tag) Validate() error {
	return validator.Validate(t)
}


func (t Tag) Encode() ([]byte, error){
	key := []byte(t.Key)
	val := []byte(t.Value)

	keyLen := len(key)
	ValueLen := len(val)

	header := new(TagHeader)
	header.KeyLen = int64(keyLen)
	header.ValueLen = int64(ValueLen)

	encodedHeader, err  := header.Encode()

	if err != nil{
		return []byte{}, nil
	}

	start := 0
	end := len(encodedHeader)

	r := make([]byte, len(encodedHeader) + len(key) + len(val))
	copy(r[start:end], encodedHeader)

	start = end
	end = start + len(key)
	copy(r[start:end], key)

	start = end
	end = start + len(val)
	copy(r[start:end], val)

	return r, nil
}

func (t *Tag) Decode(data []byte) error{
	start := int64(0)
	end := int64(16)

	header := new(TagHeader)
	err := header.Decode(data[start:end])

	if err != nil{
		return err
	}

	start = end
	end = start + header.KeyLen
	t.Key = string(data[start:end])

	start = end
	end = start +  header.ValueLen
	t.Value = string(data[start:end])

	return nil
}

// Tags
type Tags struct{
	Tags []Tag
}

// 2 bytes (totalLength)
// totalLength * (2bytes) each 2 bytes represent tag length
// tags bytes
func (t Tags) Encode() ([]byte, error){
	totalLength := len(t.Tags)
	tagsSize := 0
	tags := [][]byte{}

	for _, v := range t.Tags{
		bytes, err := v.Encode()

		if err != nil{
			return []byte{}, err
		}
		tagsSize += len(bytes)
		tags = append(tags, bytes)
	}



	r := make([]byte, 2 + tagsSize + totalLength*2)

	start := 0
	end := start + 2

	binary.LittleEndian.PutUint16(r[start:end], uint16(totalLength))

	for _, v := range tags{
		start = end
		end = start + 2
		binary.LittleEndian.PutUint16(r[start:end], uint16(len(v)))
	}

	for _, v := range tags{
		start = end
		end = start + len(v)
		copy(r[start:end], v)
	}

	return r, nil
}

func (t *Tags) Decode(data []byte) error{

	start := 0
	end := 2

	length := int16(binary.LittleEndian.Uint16(data[start:end]))
	var sizes []int16
	var result []Tag

	for i:= 0; i < int(length); i++{
		start = end
		end = start + 2
		sizes= append(sizes, int16(binary.LittleEndian.Uint16(data[start:end])))
	}

	for _, size := range sizes{
		start = end
		end = start + int(size)
		tag := new(Tag)
		err := tag.Decode(data[start:end])
		if err != nil{
			return err
		}
		result = append(result, *tag)
	}

	t.Tags = result

	return nil
}