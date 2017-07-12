package main

import (
	log "github.com/Sirupsen/logrus"
	"net/http"

	"fmt"
	"github.com/gorilla/mux"
)


// NamespacesAPI is API implementation of /namespaces root endpoint
type NamespacesAPI struct {
	apiManager     *APIManager
	config *Settings
	db DB
}

func NewNamespacesAPI(db DB, conf *Settings, apiMan *APIManager) *NamespacesAPI{
	return &NamespacesAPI{
		apiManager: apiMan,
		db: db,
		config: conf,
	}

}

func (api NamespacesAPI) DB() DB{
	return api.db
}

func (api NamespacesAPI) Config() *Settings{
	return api.config
}

func (api NamespacesAPI) APIManager() *APIManager{
	return api.apiManager
}

func (api NamespacesAPI) UpdateNamespaceStats(nsid string) error {
	nsStats := NamespaceStats{Namespace:nsid}

	_, err := nsStats.Get(api.db, api.config)
	if err != nil{
		log.Errorln(err.Error())
		return err
	}

	nsStats.NrRequests += 1

	if err := nsStats.Save(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		return err
	}
	return nil
}

func (api NamespacesAPI) GetNamespaceID(r *http.Request) string{
	return fmt.Sprintf("%s%s", api.config.Namespace.Prefix , mux.Vars(r)["nsid"])
}