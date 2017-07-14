package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"strconv"
	"github.com/zero-os/0-stor/store/rest/models"
	"github.com/zero-os/0-stor/store/db"
)

// Createobject is the handler for POST /namespaces/{nsid}/objects
// Set an object into the namespace
func (api NamespacesAPI) Createobject(w http.ResponseWriter, r *http.Request) {
	var reqBody models.Object

	nsid := mux.Vars(r)["nsid"]

	ns := models.NamespaceCreate{
		Label: nsid,
	}

	exists, err := api.db.Exists(ns.Key())

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	// decode request
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := reqBody.Validate(); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Make sure file contents are valid
	file, err := reqBody.ToFile(nsid)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	reservation := r.Context().Value("reservation").(*models.Reservation)

	oldFile := models.File{
		Id: reqBody.Id,
		Namespace: nsid,
	}

	b, err := api.db.Get(oldFile.Key())

	if err != nil {
		if err != db.ErrNotFound {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}else{
			// New file created as oldFile not exists
			if reservation.SizeRemaining() < file.Size(){
				http.Error(w, "File SizeAvailable exceeds the remaining free space in namespace", http.StatusForbidden)
				return
			}

			b, err = file.Encode()

			if err != nil{
				log.Errorln(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if err = api.db.Set(file.Key(), b); err != nil {
				log.Errorln(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			reservation.SizeUsed += file.Size()

			b, err = reservation.Encode()

			if err != nil{
				log.Errorln(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if err:= api.db.Set(reservation.Key(), b); err != nil{
				log.Errorln(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
	}else{
		err = oldFile.Decode(b)
		if err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Only update reference -- we don't update content here
		if oldFile.Reference < 255 {
			oldFile.Reference = oldFile.Reference + 1
			log.Debugln(file.Reference)
			b, err = oldFile.Encode()
			if err != nil{
				log.Errorln(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if err = api.db.Set(oldFile.Key(), b); err != nil {
				log.Errorln(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&reqBody)
}


// DeleteObject is the handler for DELETE /namespaces/{nsid}/objects/{id}
// Delete object from the KV
func (api NamespacesAPI) DeleteObject(w http.ResponseWriter, r *http.Request) {
	nsid := mux.Vars(r)["nsid"]

	ns := models.NamespaceCreate{
		Label: nsid,
	}

	exists, err := api.db.Exists(ns.Key())

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

	f := models.File{
		Namespace:nsid,
		Id: id,
	}

	v, err := api.db.Get(f.Key())

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

	err = api.db.Delete(f.Key())

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	f = models.File{}
	if err := f.Decode(v); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}


	res := r.Context().Value("reservation").(*models.Reservation)
	res.SizeUsed -= f.Size()

	b, err := res.Encode()

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if err := api.db.Set(res.Key(), b); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 204 has no bddy
	http.Error(w, "", http.StatusNoContent)
}

// GetObject is the handler for GET /namespaces/{nsid}/objects/{id}
// Retrieve object from the KV
func (api NamespacesAPI) GetObject(w http.ResponseWriter, r *http.Request) {

	nsid := mux.Vars(r)["nsid"]

	ns := models.NamespaceCreate{
		Label: nsid,
	}

	exists, err := api.db.Exists(ns.Key())

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

	f := models.File{
		Namespace:nsid,
		Id: id,
	}

	v, err := api.db.Get(f.Key())

	if err != nil {
		if err == db.ErrNotFound{
			http.Error(w, "object doesn't exist", http.StatusNotFound)
			return
		}
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}


	err = f.Decode(v)

	// Database Error
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}


	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&models.Object{
		Id: f.Id,
		Tags: []models.Tag{},
		Data: string(f.Payload),
	})
}

// HeadObject is the handler for HEAD /namespaces/{nsid}/objects/{id}
// Tests object exists in the KV
func (api NamespacesAPI) HeadObject(w http.ResponseWriter, r *http.Request) {
	nsid := mux.Vars(r)["nsid"]

	ns := models.NamespaceCreate{
		Label: nsid,
	}

	exists, err := api.db.Exists(ns.Key())

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

	f := models.File{
		Namespace:nsid,
		Id: id,
	}


	exists, err = api.db.Exists(f.Key())

	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if exists {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

// Listobjects is the handler for GET /namespaces/{nsid}/objects
// List keys of the namespaces
func (api NamespacesAPI) Listobjects(w http.ResponseWriter, r *http.Request) {
	var respBody []models.Object

	// Pagination
	pageParam := r.FormValue("page")
	per_pageParam := r.FormValue("per_page")

	if pageParam == "" {
		pageParam = "1"
	}

	if per_pageParam == "" {
		per_pageParam = "20"
	}

	page, err := strconv.Atoi(pageParam)

	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	per_page, err := strconv.Atoi(per_pageParam)

	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	nsid := mux.Vars(r)["nsid"]


	ns := models.NamespaceCreate{
		Label: nsid,
	}

	exists, err := api.db.Exists(ns.Key())

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	startingIndex := (page-1)*per_page + 1
	resultsCount := per_page

	prefixStr := fmt.Sprintf("%s:", nsid)

	objects, err := api.db.Filter(prefixStr, startingIndex, resultsCount)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	respBody = make([]models.Object, 0, len(objects))

	for _, record := range(objects){
		f := new(models.File)
		if err := f.Decode(record); err != nil {
			log.Errorln("Error decoding namespace :%v", err)
			http.Error(w, "Error decoding namespace", http.StatusInternalServerError)
			return
		}

		o := models.Object{
			Id: f.Id,
			Tags: []models.Tag{},
			Data: string(f.Payload),
		}

		respBody = append(respBody, o)
	}

	// return empty list if no results
	if len(respBody) == 0 {
		respBody = []models.Object{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}

// UpdateObject is the handler for PUT /namespaces/{nsid}/objects/{id}
// Update oject
func (api NamespacesAPI) UpdateObject(w http.ResponseWriter, r *http.Request) {
	var reqBody models.ObjectUpdate

	// decode request
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := reqBody.Validate(); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	nsid := mux.Vars(r)["nsid"]

	// Make sure file contents are valid
	file, err := reqBody.ToFile(nsid)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}


	id := mux.Vars(r)["id"]

	oldFile := models.File{
		Id: id,
		Namespace: nsid,
	}

	b, err := api.db.Get(oldFile.Key())

	if err != nil {
		if err == db.ErrNotFound{
			http.Error(w, "Object doesn't exist", http.StatusNotFound)
			return
		}else {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

	err = oldFile.Decode(b)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Prepend the same value of the first byte of old data
	file.Reference = oldFile.Reference

	b, err = file.Encode()
	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// Add object
	if err = api.db.Set(oldFile.Key(), b); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	res := r.Context().Value("reservation").(models.Reservation)

	diff := oldFile.Size() - file.Size()

	res.SizeUsed += diff

	b, err = res.Encode()

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err:= api.db.Set(res.Key(), b); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&models.Object{
		Id:   id,
		Data: reqBody.Data,
		Tags: reqBody.Tags,
	})
}
