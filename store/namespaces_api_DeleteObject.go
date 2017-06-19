package main

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/librairies/reservation"

)

// DeleteObject is the handler for DELETE /namespaces/{nsid}/objects/{id}
// Delete object from the store
func (api NamespacesAPI) DeleteObject(w http.ResponseWriter, r *http.Request) {
	namespace := mux.Vars(r)["nsid"]
	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", namespace, id)

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


	statsBytes := r.Context().Value("stats").([]byte)
	stats := Stat{}
	stats.fromBytes(statsBytes)
	statsKey := r.Context().Value("statsKey").(string)

	reservationBytes := r.Context().Value("reservation").([]byte)
	res := reservation.Reservation{}
	res.FromBytes(reservationBytes)
	reservationKey := r.Context().Value("reservationKey").(string)

	stats.SizeUsed -= int64(f.Size())
	res.SizeUsed -= int64(f.Size())

	if err:= api.db.Set(statsKey, stats.toBytes()); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err:= api.db.Set(reservationKey, res.ToBytes()); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}


	// 204 has no bddy
	http.Error(w, "", http.StatusNoContent)
}
