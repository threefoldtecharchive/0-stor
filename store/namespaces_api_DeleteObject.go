package main

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

// DeleteObject is the handler for DELETE /namespaces/{nsid}/objects/{id}
// Delete object from the store
func (api NamespacesAPI) DeleteObject(w http.ResponseWriter, r *http.Request) {
	nsid := mux.Vars(r)["nsid"]

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	prefixedNamespaceID := fmt.Sprintf("%s%s", api.config.Namespace.prefix, nsid)
	exists, err := api.db.Exists(prefixedNamespaceID)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", nsid, id)

	v, err := api.db.Get(key)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if v == nil {
		http.Error(w, "Namespace or object doesn't exist", http.StatusNotFound)
		return
	}

	err = api.db.Delete(key)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	f := File{}
	f.FromBytes(v)


	res := r.Context().Value("reservation").(*Reservation)
	res.SizeUsed -= f.Size()

	if err:= res.Save(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 204 has no bddy
	http.Error(w, "", http.StatusNoContent)
}
