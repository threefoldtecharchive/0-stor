package main

import (
	"net/http"
	"github.com/gorilla/mux"
	"github.com/zaibon/badger/badger"
	"log"
	"strings"
	"fmt"
)

// Deletensid is the handler for DELETE /namespaces/{nsid}
// Delete nsid
func (api NamespacesAPI) Deletensid(w http.ResponseWriter, r *http.Request) {
	nsid := mux.Vars(r)["nsid"]

	v, err :=  api.db.Get(nsid)

	if err != nil{
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// NOT FOUND
	if v == nil{
		http.Error(w, "Namespace doesn't exist", http.StatusNotFound)
		return
	}

	err2 := api.db.Delete(nsid)

	if err2 != nil{
		log.Println(err2.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Delete objects in a namespace
	defer func(){
		opt := badger.IteratorOptions{}
		opt.FetchValues = api.config.Iterator.FetchValues
		opt.PrefetchSize = api.config.Iterator.FetchSize
		opt.Reverse = false

		prefix := fmt.Sprintf("%s:", nsid)

		it := api.db.store.NewIterator(opt)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key()[:])
			if !strings.Contains(key, prefix){
				continue
			}

			if err := api.db.Delete(key); err != nil{
				log.Println(err.Error())

			}
		}
	}()

	http.Error(w, "", http.StatusNoContent)
}
