package crypto

import (
	"fmt"
	"strings"
)

// Sum256 create and returns a hash,
// for and given some binary input data,
// using the default hashing algorithm, SHA256.
func Sum256(data []byte) (hash []byte) {
	return SumSHA256(data)
}

// Sum512 create and returns a hash,
// for and given some binary input data,
// using the default hashing algorithm, SHA512.
func Sum512(data []byte) (hash []byte) {
	return SumSHA512(data)
}

// NewDefaultHasher256 returns a new instance of the default hasher type.
//
// Key is an optional private key to add authentication to the output,
// when the key is not given the hasher will produce
// cryptographically secure checksums, without any proof of ownership.
//
// The default hasher is currently SHA256, which produces checksums of 32 bytes.
//
// This package reserves the right to change the
// default 256 bit hashing algorithm at any time,
// but this constructor will always be available and up to date.
func NewDefaultHasher256(key []byte) (Hasher, error) {
	return NewSHA256Hasher(key)
}

// NewDefaultHasher512 returns a new instance of the default hasher type.
//
// Key is an optional private key to add authentication to the output,
// when the key is not given the hasher will produce
// cryptographically secure checksums, without any proof of ownership.
//
// The default hasher is currently SHA512, which produces checksums of 64 bytes.
//
// This package reserves the right to change the
// default 512 bit hashing algorithm at any time,
// but this constructor will always be available and up to date.
func NewDefaultHasher512(key []byte) (Hasher, error) {
	return NewSHA512Hasher(key)
}

// NewHasher returns a new instance for the given hasher type.
// If the hasher type is invalid, an error is returned.
//
// Key is an optional private key to add authentication to the output,
// when the key is not given the hasher will produce
// cryptographically secure checksums, without any proof of ownership.
func NewHasher(ht HashType, key []byte) (Hasher, error) {
	hc, ok := _HashTypeValueToConstructorMapping[ht]
	if !ok {
		return nil, fmt.Errorf("%d is not a valid HashType value", ht)
	}
	return hc(key)
}

// HashFunc create and returns a hash,
// for and given some binary input data.
type HashFunc func(data []byte) (hash []byte)

// Hasher defines the interface of a crypto-hasher,
// which can be used to create a hash, given some binary input data.
//
// Hasher is /NEVER/ thread-safe,
// and should only ever be used on /ONE/ goroutine at a time.
// If you wish to hash on multiple goroutines,
// you'll have to create one Hasher per goroutine.
type Hasher interface {
	// HashBytes creates a secure hash,
	// given some input data.
	HashBytes(data []byte) (hash []byte)
}

// HashType represents a cryptographic hashing algorithm.
type HashType uint8

const (
	// HashTypeSHA256 is the enum value which identifiers SHA256,
	// a cryptographic hashing algorithm which produces a secure hash of 32 bytes.
	// This type is also the default HashType.
	HashTypeSHA256 HashType = iota
	// HashTypeSHA512 is the enum value which identifiers SHA512,
	// a cryptographic hashing algorithm which produces a secure hash of 64 bytes.
	HashTypeSHA512
	// HashTypeBlake2b256 is the enum value which identifiers Blake2b-256,
	// a cryptographic hashing algorithm which produces a secure hash of 32 bytes.
	HashTypeBlake2b256
	// HashTypeBlake2b512 is the enum value which identifiers Blake2b-512,
	// a cryptographic hashing algorithm which produces a secure hash of 64 bytes.
	HashTypeBlake2b512

	// DefaultHash256Type represents the default 256 bit
	// Hashing algorithm as promoted by this package.
	//
	// This package reserves the right to change the
	// default 256 bit hashing algorithm at any time,
	// but this constant will always be available and up to date.
	DefaultHash256Type = HashTypeSHA256

	// DefaultHash512Type represents the default 512 bit
	// Hashing algorithm as promoted by this package.
	//
	// This package reserves the right to change the
	// default 512 bit hashing algorithm at any time,
	// but this constant will always be available and up to date.
	DefaultHash512Type = HashTypeSHA512

	// MaxStandardHashType defines the hasher type,
	// which has the greatest defined/used enum value.
	// When defining your custom HashType you can do so as follows:
	//
	//    const (
	//         MyHashType = iota + MaxStandardHashType + 1
	//         MyOtherHashType
	//         // ...
	//    )
	//
	// The maximum allowed value of a custom hash type is 255,
	// due to the underlying uint8 type.
	MaxStandardHashType = HashTypeBlake2b512
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

// HasherConstructor defines a function which can be used to create a hasher.
// The key parameter is an optional private key which can be used,
// to create signatures as to provide authentication.
type HasherConstructor func(key []byte) (Hasher, error)

// RegisterHash registers a new or overwrite an existing hash algorithm.
// The given str will be used in a case-insensitive manner,
// if the registered hash however overwrites an existing hash type,
// the str parameter will be ignored and instead the already used string version will be used.
// This is intended to be called from the init function in packages that implement hash functions.
func RegisterHash(ht HashType, str string, hc HasherConstructor) {
	if hc == nil {
		panic("no hash constructor given")
	}

	if s, ok := _HashTypeValueToStringMapping[ht]; ok {
		str = s // ignoring given string
	} else if str == "" {
		panic("no string version defined for new hash type")
	}

	_HashTypeValueToStringMapping[ht] = str
	_HashTypeStringToValueMapping[str] = ht
	_HashTypeValueToConstructorMapping[ht] = hc
}

// hashing algorithms mapping used to create hasher instances,
// based on their enum value, as well as to
// convert between the string and enum type values.
var (
	_HashTypeValueToStringMapping      = make(map[HashType]string)
	_HashTypeStringToValueMapping      = make(map[string]HashType)
	_HashTypeValueToConstructorMapping = make(map[HashType]HasherConstructor)
)
