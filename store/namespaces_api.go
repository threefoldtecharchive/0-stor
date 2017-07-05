package main

import (
	log "github.com/Sirupsen/logrus"
	"net/http"

	"fmt"
	"github.com/gorilla/mux"
)


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
	return fmt.Sprintf("%s%s", api.config.Namespace.prefix , mux.Vars(r)["nsid"])
}