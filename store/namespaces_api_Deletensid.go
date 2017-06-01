package main

import (
	"net/http"
	"github.com/gorilla/mux"
)

// Deletensid is the handler for DELETE /namespaces/{nsid}
// Delete nsid
func (api NamespacesAPI) Deletensid(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["nsid"]
	api.db.Delete(key)
	w.WriteHeader(204)
}
