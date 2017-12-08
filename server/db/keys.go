package db

import "fmt"

// DataPrefix returns the data prefix for a given label.
func DataPrefix(label []byte) []byte {
	if label == nil {
		panic("no label given")
	}

	return []byte(fmt.Sprintf("%s:%s", label, PrefixData))
}

// DataKey returns the data key for a given label and key.
func DataKey(label, key []byte) []byte {
	if label == nil {
		panic("no label given")
	}
	if key == nil {
		panic("no key given")
	}

	return []byte(fmt.Sprintf("%s:%s:%s", label, PrefixData, key))
}

// DataKeyPrefixLength returns the length of the prefix of a data key.
// That is to say, the total length of a data key
// minus the length of the object key.
func DataKeyPrefixLength(label []byte) int {
	if label == nil {
		panic("no label given")
	}

	return len(label) + prefixDataLength + 2 // 2 -> seperators
}

// ReferenceListPrefix returns the reference list prefix for a given label.
func ReferenceListPrefix(label []byte) []byte {
	if label == nil {
		panic("no label given")
	}

	return []byte(fmt.Sprintf("%s:%s", label, PrefixReferenceList))
}

// ReferenceListKey returns the reference list key for a given label and key.
func ReferenceListKey(label, key []byte) []byte {
	if label == nil {
		panic("no label given")
	}
	if key == nil {
		panic("no key given")
	}

	return []byte(fmt.Sprintf("%s:%s:%s", label, PrefixReferenceList, key))
}

// ReferenceListKeyPrefixLength returns the length of the prefix of a reference list key.
// That is to say, the total length of a reference list key
// minus the length of the object key.
func ReferenceListKeyPrefixLength(label []byte) int {
	if label == nil {
		panic("no label given")
	}

	return len(label) + prefixReferenceListLength + 2 // 2 -> seperators
}

// NamespaceKey returns the label key for a given label.
func NamespaceKey(label []byte) []byte {
	if label == nil {
		panic("no label given")
	}

	return []byte(fmt.Sprintf("%s:%s", PrefixNamespace, label))
}

const (
	// PrefixData is the prefix to be used to store data (blobs).
	PrefixData = "d"
	// PrefixReferenceList is the prefix to be used to store reference list(s).
	PrefixReferenceList = "rl"
	// PrefixNamespace is  the prefix to be used to store namespaces
	PrefixNamespace = "@"
)

const (
	prefixDataLength          = len(PrefixData)
	prefixReferenceListLength = len(PrefixReferenceList)
)

const (
	// KeyStoreStats is the key (name) to be used to store
	// the global store statistics.
	KeyStoreStats = "$"
)
