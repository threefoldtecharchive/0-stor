package crypto

import (
	"fmt"
	"strings"
)

// HashBytes create and returns a hash,
// for and given some binary input data,
// using the default hashing algorithm, SHA256.
func HashBytes(data []byte) (hash []byte) {
	return SHA256Hash(data)
}

// NewHasher returns a new instance of the default hasher type.
// The default hasher is SHA256, which produces checksums of 32 bytes.
func NewHasher() (Hasher, error) {
	return NewSHA256hasher()
}

// NewHasherForType returns a new instance for the given hasher type.
// If the hasher type is invalid, an error is returned.
func NewHasherForType(ht HashType) (Hasher, error) {
	switch ht {
	case HashTypeSHA256:
		return NewSHA256hasher()

	case HashTypeSHA512:
		return NewSHA512hasher()

	case HashTypeBlake2b:
		return NewBlake2bHasher(nil)

	default:
		return nil, fmt.Errorf("%d is not a valid HashType value", ht)
	}
}

// HashFunc create and returns a hash,
// for and given some binary input data.
type HashFunc func(data []byte) (hash []byte)

// Hasher defines the interface of a crypto-hasher,
// which can be used to create a hash, given some binary input data.
type Hasher interface {
	// HashBytes creates a secure hash,
	// given some input data.
	HashBytes(data []byte) (hash []byte)
}

// HashType represents
type HashType uint8

const (
	// HashTypeSHA256 is the enum value which identifiers SHA256,
	// a cryptographic hashing algorithm which produces a secure hash of 32 bytes.
	// This type is also the default HashType.
	HashTypeSHA256 HashType = iota
	// HashTypeSHA512 is the enum value which identifiers SHA512,
	// a cryptographic hashing algorithm which produces a secure hash of 64 bytes.
	HashTypeSHA512
	// HashTypeBlake2b is the enum value which identifiers Blake2b,
	// a cryptographic hashing algorithm which produces a secure hash of 32 bytes.
	HashTypeBlake2b
)

// String implements Stringer.String
func (ht HashType) String() string {
	str, ok := _HashTypeValueToStringMapping[ht]
	if !ok {
		return ""
	}
	return str
}

// MarshalText implements encoding.TextMarshaler.MarshalText
func (ht HashType) MarshalText() ([]byte, error) {
	str := ht.String()
	if str == "" {
		return nil, fmt.Errorf("%d is not a valid HashType value", ht)
	}
	return []byte(str), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText
func (ht *HashType) UnmarshalText(text []byte) error {
	var ok bool
	*ht, ok = _HashTypeStringToValueMapping[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("%q is not a valid HashType string", text)
	}
	return nil
}

// variables used to map between the HashType enum values
// and its string representation.

const _HashTypeStrings = "sha_256sha_512blake2b_256"

var _HashTypeValueToStringMapping = map[HashType]string{
	HashTypeSHA256:  _HashTypeStrings[:7],
	HashTypeSHA512:  _HashTypeStrings[7:14],
	HashTypeBlake2b: _HashTypeStrings[14:],
}
var _HashTypeStringToValueMapping = map[string]HashType{
	_HashTypeStrings[:7]:    HashTypeSHA256,
	_HashTypeStrings[:3]:    HashTypeSHA256,
	_HashTypeStrings[7:14]:  HashTypeSHA512,
	_HashTypeStrings[14:]:   HashTypeBlake2b,
	_HashTypeStrings[14:21]: HashTypeBlake2b,
}
