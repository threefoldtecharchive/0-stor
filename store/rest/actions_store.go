package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/rest/models"
)

// statsPost is the handler for POST /namespaces/stats
// Update Global Store statistics and available space
func (api NamespacesAPI) UpdateStoreStats(w http.ResponseWriter, r *http.Request) {
	var reqBody models.StoreStatRequest

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := reqBody.Validate(); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	storeStat := new(models.StoreStat)
	b, err := api.db.Get(storeStat.Key())
	if err != nil {
		if err == db.ErrNotFound {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		} else {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	}

	err = storeStat.Decode(b)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if reqBody.SizeAvailable < storeStat.SizeUsed {
		err := fmt.Sprintf("Store stats available size must be greater than used size (%v)", storeStat.SizeUsed)
		http.Error(w, err, http.StatusBadRequest)
		return
	}

	storeStat.SizeAvailable = reqBody.SizeAvailable

	b, err = storeStat.Encode()

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := api.db.Set(storeStat.Key(), b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&reqBody)
}

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(&respBody)
}
