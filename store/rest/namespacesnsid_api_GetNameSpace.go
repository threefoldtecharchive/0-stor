package rest

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/manager"
)

// GetNameSpace is the handler for GET /namespaces/{nsid}
// Get detail view about namespace
func (api NamespacesAPI) GetNameSpace(w http.ResponseWriter, r *http.Request) {
	label := mux.Vars(r)["nsid"]

	mgr := manager.NewNamespaceManager(api.db)

	ns, err := mgr.Get(label)
	if err != nil {
		if err == db.ErrNotFound {
			jsonError(w, err, http.StatusNotFound)
		} else {
			jsonError(w, err, http.StatusInternalServerError)
		}
		return
	}

	respBody := Namespace{
		Label: ns.Label,
		Stats: NamespaceStat{
		// TODO
		// NrObjects
		// RequestPerHour
		// SpaceAvailable:
		// SpaceUsed:
		},
	}
	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}
