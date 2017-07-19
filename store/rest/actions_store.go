package rest

import (
	"encoding/json"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-units"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/rest/models"
)

// GetStoreStats is the handler for GET /namespaces/stats
// Return usage statistics about the whole KV
func (api NamespacesAPI) GetStoreStats(w http.ResponseWriter, r *http.Request) {
	respBody := new(models.StoreStat)

	b, err := api.db.Get(respBody.Key())
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = respBody.Decode(b)
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody.SizeAvailable /= units.MiB
	respBody.SizeUsed /= units.MiB

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(&respBody)
}
