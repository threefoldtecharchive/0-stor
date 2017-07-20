package rest

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/manager"
	"github.com/zero-os/0-stor/store/stats"
)

// DeleteObject is the handler for DELETE /namespaces/{nsid}/objects/{id}
// Delete object from the store
func (api NamespacesAPI) DeleteObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["nsid"]
	key := []byte(vars["id"])

	// increase request counter
	go stats.IncrWrite(mux.Vars(r)["nsid"])

	mgr := manager.NewObjectManager(namespace, api.db)

	if err := mgr.Delete(key); err != nil {
		if err == db.ErrNotFound {
			jsonError(w, err, http.StatusNotFound)
		} else {
			jsonError(w, err, http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
