package db

import "fmt"

// DataPrefix returns the data prefix for a given namespace.
func DataPrefix(namespace []byte) []byte {
	if namespace == nil {
		panic("no namespace given")
	}

	return []byte(fmt.Sprintf("%s:%s", namespace, PrefixData))
}

// DataKey returns the data key for a given namespace and key.
func DataKey(namespace, key []byte) []byte {
	if namespace == nil {
		panic("no namespace given")
	}
	if key == nil {
		panic("no key given")
	}

	return []byte(fmt.Sprintf("%s:%s:%s", namespace, PrefixData, key))
}

// ReferenceListPrefix returns the reference list prefix for a given namespace.
func ReferenceListPrefix(namespace []byte) []byte {
	if namespace == nil {
		panic("no namespace given")
	}

	return []byte(fmt.Sprintf("%s:%s", namespace, PrefixReferenceList))
}

// ReferenceListKey returns the reference list key for a given namespace and key.
func ReferenceListKey(namespace, key []byte) []byte {
	if namespace == nil {
		panic("no namespace given")
	}
	if key == nil {
		panic("no key given")
	}

	return []byte(fmt.Sprintf("%s:%s:%s", namespace, PrefixReferenceList, key))
}

// NamespaceKey returns the namespace key for a given namespace.
func NamespaceKey(namespace []byte) []byte {
	if namespace == nil {
		panic("no namespace given")
	}

	return []byte(fmt.Sprintf("%s:%s", PrefixNamespace, namespace))
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
	// KeyStoreStats is the key (name) to be used to store
	// the global store statistics.
	KeyStoreStats = "$"
)
