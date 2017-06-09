package main

// NamespacesAPI is API implementation of /namespaces root endpoint
type NamespacesAPI struct {
	db     *Badger
	config *settings
}
