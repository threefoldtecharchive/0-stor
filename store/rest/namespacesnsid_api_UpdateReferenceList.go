package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/manager"
)

// UpdateReferenceList is the handler for PUT /namespaces/{nsid}/objects/{id}/references
// Update reference list.
// The reference list of the object will be update with the references from the request body
func (api NamespacesAPI) UpdateReferenceList(w http.ResponseWriter, r *http.Request) {
	var reqBody []ReferenceID

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		jsonError(w, err, http.StatusBadRequest)
		return
	}

	if len(reqBody) > 160 {
		jsonError(w, fmt.Errorf("reference list is too big, can only contains 160 entries"), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	namespace := vars["nsid"]
	key := []byte(vars["id"])

	mgr := manager.NewObjectManager(namespace, api.db)

	obj, err := mgr.Get(key)
	if err != nil {
		if err == db.ErrNotFound {
			jsonError(w, err, http.StatusNotFound)
		} else {
			jsonError(w, err, http.StatusInternalServerError)
		}
		return
	}
	refList := [160][]byte{}
	for i := range reqBody {
		refList[i] = []byte(reqBody[i])
	}

	if err := mgr.Set([]byte(key), obj.Data, refList); err != nil {
		jsonError(w, err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&reqBody)
}