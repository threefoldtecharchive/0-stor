package main

import (
	"net/http"
)

type DataTokenValidMiddleware struct{
	acl ACLEntry
}

func NewDataTokenValidMiddleware(acl ACLEntry) *DataTokenValidMiddleware{
	return &DataTokenValidMiddleware{
		acl : acl,
	}
}

func (dt *DataTokenValidMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("data-access-token")
		if token == ""{
			http.Error(w, "Data access token is missing", http.StatusUnauthorized)
			return
		}

		res := Reservation{}

		if err := res.ValidateDataAccessToken(dt.acl, token); err != nil{
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
