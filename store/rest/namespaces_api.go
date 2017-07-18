package rest

import (
	"github.com/zero-os/0-stor/store/config"
	"github.com/zero-os/0-stor/store/db"
)

var _ (NamespacesInterface) = (*NamespacesAPI)(nil)

// NamespacesAPI is API implementation of /namespaces root endpoint
type NamespacesAPI struct {
	config config.Settings
	db     db.DB
}

func NewNamespacesAPI(db db.DB, conf config.Settings) *NamespacesAPI {
	return &NamespacesAPI{
		db:     db,
		config: conf,
	}

}

func (api NamespacesAPI) DB() db.DB {
	return api.db
}

func (api NamespacesAPI) Config() config.Settings {
	return api.config
}
