package main

import (
	"encoding/json"
	"net/http"
)

// NamespacesAPI is API implementation of /namespaces root endpoint
type NamespacesAPI struct {
}

// nsidaclPost is the handler for POST /namespaces/{nsid}/acl
// Create an dataAccessToken for a user. This token gives this user access to the data in this namespace
func (api NamespacesAPI) nsidaclPost(w http.ResponseWriter, r *http.Request) {
	var reqBody ACL

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

	// validate request
	if err := reqBody.Validate(); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// DeleteObject is the handler for DELETE /namespaces/{nsid}/objects/{id}
// Delete object from the store
func (api NamespacesAPI) DeleteObject(w http.ResponseWriter, r *http.Request) {
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// GetObject is the handler for GET /namespaces/{nsid}/objects/{id}
// Retrieve object from the store
func (api NamespacesAPI) GetObject(w http.ResponseWriter, r *http.Request) {
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// UpdateObject is the handler for PUT /namespaces/{nsid}/objects/{id}
// Update oject
func (api NamespacesAPI) UpdateObject(w http.ResponseWriter, r *http.Request) {
	var reqBody ObjectUpdate

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

	// validate request
	if err := reqBody.Validate(); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// nsidobjectsGet is the handler for GET /namespaces/{nsid}/objects
// List keys of the namespaces
func (api NamespacesAPI) nsidobjectsGet(w http.ResponseWriter, r *http.Request) {
	var respBody []Object
	json.NewEncoder(w).Encode(&respBody)
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// nsidobjectsPost is the handler for POST /namespaces/{nsid}/objects
// Set an object into the namespace
func (api NamespacesAPI) nsidobjectsPost(w http.ResponseWriter, r *http.Request) {
	//var reqBody ObjectCreate
	//
	//// decode request
	//if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
	//	w.WriteHeader(400)
	//	return
	//}
	//
	//// validate request
	//if err := reqBody.Validate(); err != nil {
	//	w.WriteHeader(400)
	//	w.Write([]byte(`{"error":"` + err.Error() + `"}`))
	//	return
	//}
	//var respBody Object
	//json.NewEncoder(w).Encode(&respBody)
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// nsidreservationidGet is the handler for GET /namespaces/{nsid}/reservation/{id}
// Return information about a reservation
func (api NamespacesAPI) nsidreservationidGet(w http.ResponseWriter, r *http.Request) {
	var reqBody Reservation

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

	// validate request
	if err := reqBody.Validate(); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// nsidreservationidPut is the handler for PUT /namespaces/{nsid}/reservation/{id}
// Renew an existing reservation
func (api NamespacesAPI) nsidreservationidPut(w http.ResponseWriter, r *http.Request) {
	var reqBody ReservationRequest

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

	// validate request
	if err := reqBody.Validate(); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// nsidreservationGet is the handler for GET /namespaces/{nsid}/reservation
// Return a list of all the existing reservation for the give resource
func (api NamespacesAPI) nsidreservationGet(w http.ResponseWriter, r *http.Request) {
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// nsidreservationPost is the handler for POST /namespaces/{nsid}/reservation
// Create a reservation for the given resource.
func (api NamespacesAPI) nsidreservationPost(w http.ResponseWriter, r *http.Request) {
	var reqBody ReservationRequest

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

	// validate request
	if err := reqBody.Validate(); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}
	var respBody NamespacesNsidReservationPostRespBody
	json.NewEncoder(w).Encode(&respBody)
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// StatsNamespace is the handler for GET /namespaces/{nsid}/stats
// Return usage statistics about this namespace
func (api NamespacesAPI) StatsNamespace(w http.ResponseWriter, r *http.Request) {
	var respBody NamespaceStat
	json.NewEncoder(w).Encode(&respBody)
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// nsidDelete is the handler for DELETE /namespaces/{nsid}
// Delete nsid
func (api NamespacesAPI) nsidDelete(w http.ResponseWriter, r *http.Request) {
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// nsidGet is the handler for GET /namespaces/{nsid}
// Get detail view about nsid
func (api NamespacesAPI) nsidGet(w http.ResponseWriter, r *http.Request) {
	//var respBody Nsid
	var respBody string
	json.NewEncoder(w).Encode(&respBody)
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// nsidPut is the handler for PUT /namespaces/{nsid}
// Update nsid
func (api NamespacesAPI) nsidPut(w http.ResponseWriter, r *http.Request) {
	//var respBody Nsid
	var respBody string
	json.NewEncoder(w).Encode(&respBody)
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// Get is the handler for GET /namespaces
// List all namespaces
func (api NamespacesAPI) Get(w http.ResponseWriter, r *http.Request) {
	var respBody []Namespace
	json.NewEncoder(w).Encode(&respBody)
	// uncomment below line to add header
	// w.Header().Set("key","value")
}

// Post is the handler for POST /namespaces
// Create a new namespace
func (api NamespacesAPI) Post(w http.ResponseWriter, r *http.Request) {
	var reqBody NamespaceCreate

	// decode request
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(400)
		return
	}

	// validate request
	if err := reqBody.Validate(); err != nil {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"` + err.Error() + `"}`))
		return
	}
	var respBody Namespace
	json.NewEncoder(w).Encode(&respBody)
	// uncomment below line to add header
	// w.Header().Set("key","value")
}
