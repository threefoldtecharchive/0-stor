package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/dgraph-io/badger"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"strings"
	"strconv"
)

// Createobject is the handler for POST /namespaces/{nsid}/objects
// Set an object into the namespace
func (api NamespacesAPI) Createobject(w http.ResponseWriter, r *http.Request) {
	var reqBody Object

	nsid := mux.Vars(r)["nsid"]

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	prefixedNamespaceID := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, nsid)
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
	file, err := reqBody.ToFile(true)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	reservation := r.Context().Value("reservation").(*Reservation)

	key := fmt.Sprintf("%s:%s", nsid, reqBody.Id)

	oldFile, err := api.db.GetFile(key)

	// Database Error
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// object already exists
	if oldFile != nil {
		// Only update reference -- we don't update content here
		if oldFile.Reference < 255 {
			oldFile.Reference = oldFile.Reference + 1
			log.Debugln(file.Reference)
			if err = api.db.Set(key, oldFile.Encode()); err != nil {
				log.Errorln(err.Error())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}
	}else{
		// New file created
		if reservation.SizeRemaining() < file.Size(){
			http.Error(w, "File SizeAvailable exceeds the remaining free space in namespace", http.StatusForbidden)
			return
		}

		if err = api.db.Set(key, file.Encode()); err != nil {
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		reservation.SizeUsed += file.Size()

		if err:= reservation.Save(api.db, api.config); err != nil{
			log.Errorln(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
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

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	prefixedNamespaceID := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, nsid)
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
	f.Decode(v)


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

// GetObject is the handler for GET /namespaces/{nsid}/objects/{id}
// Retrieve object from the KV
func (api NamespacesAPI) GetObject(w http.ResponseWriter, r *http.Request) {

	var file = &File{}

	oldLabel := mux.Vars(r)["nsid"]

	// Update namespace stats
	defer api.UpdateNamespaceStats(oldLabel)

	nsid := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, oldLabel)

	exists, err := api.db.Exists(nsid)

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

	key := fmt.Sprintf("%s:%s", oldLabel, id)

	value, err := api.db.Get(key)

	// Database Error
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// KEY NOT FOUND
	if value == nil {
		http.Error(w, "Object doesn't exist", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(file.ToObject(value, id))
}

// HeadObject is the handler for HEAD /namespaces/{nsid}/objects/{id}
// Tests object exists in the KV
func (api NamespacesAPI) HeadObject(w http.ResponseWriter, r *http.Request) {
	nsid := mux.Vars(r)["nsid"]

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	prefixedNamespaceID := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, nsid)
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

	exists, err = api.db.Exists(key)

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
	var respBody []Object

	// Pagination
	pageParam := r.FormValue("page")
	per_pageParam := r.FormValue("per_page")

	if pageParam == "" {
		pageParam = "1"
	}

	if per_pageParam == "" {
		per_pageParam = strconv.Itoa(api.config.DB.Pagination.PageSize)
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

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	prefixedNsid := fmt.Sprintf("%s%s", api.config.Namespace.Prefix, nsid)
	exists, err := api.db.Exists(prefixedNsid)

	if err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if !exists{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	prefixStr := fmt.Sprintf("%s:", nsid)
	prefix := []byte(prefixStr)

	opt := badger.DefaultIteratorOptions
	opt.PrefetchSize = api.config.DB.Iterator.PreFetchSize

	it := api.db.KV.NewIterator(opt)
	defer it.Close()

	startingIndex := (page-1)*per_page + 1
	counter := 0 // Number of objects encountered
	resultsCount := per_page

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		item := it.Item()
		key := string(item.Key()[:])

		// Found a namespace
		counter++

		// Skip this object if its index < intended startingIndex
		if counter < startingIndex {
			continue
		}

		value := item.Value()

		var file = &File{}
		object := file.ToObject(value, key)

		// remove prefix from file name
		object.Id = strings.Replace(object.Id, prefixStr, "", 1)
		respBody = append(respBody, *object)

		if len(respBody) == resultsCount {
			break
		}
	}

	// return empty list if no results
	if len(respBody) == 0 {
		respBody = []Object{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&respBody)
}

// UpdateObject is the handler for PUT /namespaces/{nsid}/objects/{id}
// Update oject
func (api NamespacesAPI) UpdateObject(w http.ResponseWriter, r *http.Request) {
	var reqBody ObjectUpdate

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
	file, err := reqBody.ToFile(true)

	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	nsid := mux.Vars(r)["nsid"]

	// Update namespace stats
	defer api.UpdateNamespaceStats(nsid)

	id := mux.Vars(r)["id"]

	key := fmt.Sprintf("%s:%s", nsid, id)

	oldFile, err := api.db.GetFile(key)

	// Database Error
	if err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// KEY NOT FOUND
	if oldFile == nil {
		http.Error(w, "Object doesn't exist", http.StatusNotFound)
		return
	}

	// Prepend the same value of the first byte of old data
	file.Reference = oldFile.Reference

	// Add object
	if err = api.db.Set(key, file.Encode()); err != nil {
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}

	res := r.Context().Value("reservation").(Reservation)

	diff := oldFile.Size() - file.Size()

	res.SizeUsed += diff

	if err:= res.Save(api.db, api.config); err != nil{
		log.Errorln(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&Object{
		Id:   id,
		Data: reqBody.Data,
		Tags: reqBody.Tags,
	})
}
