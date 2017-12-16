// Package crypto collects common cryptographic components.
//
// Hasher is currently the only component exposed by this package.
// A hasher can be created using a HasherType enum value (NewHasher),
// but it can also be created using NewDefaultHasher256/NewDefaultHasher512 or
// the constructor for the Hasher itself.
//
// Each enumeration value also has a string version,
// which is used by the HashType in order to implement the
// TextMarshaler and TextUnmarshaler interfaces.
//
// You can use the RegisterHasher function to
// register your own hash by giving a unique HashType enum value,
// string version and constructor.
// This will make the hash type a first-class citizen of this package.
//
// You can also overwrite an existing hash function,
// by using a HashType enum value Already used in a prior registration,
// and in this case the str input parameter doesn't have to be given,
// as the existing string version will be used in that case.
//
// Each Hasher supported by this package also has a standalone Sum
// function which can be used to create a checksum,
// without having to create a Hasher first,
// this is useful for scenarios where you only need to hash infrequently.
//
// These non-processing components are used in the 0-stor client
// If you're looking for cryptographic components
package crypto
