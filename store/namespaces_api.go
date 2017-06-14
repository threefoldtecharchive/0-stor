package main

// NamespacesAPI is API implementation of /namespaces root endpoint
type NamespacesAPI struct {
	db     *Badger
	config *settings
}

func (api NamespacesAPI) DB() *Badger{
	return api.db
}

func (api NamespacesAPI) Config() *settings{
	return api.config
}