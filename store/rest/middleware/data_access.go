package middleware

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/jwt"
	"github.com/zero-os/0-stor/store/rest/models"
)

type DataTokenValidMiddleware struct {
	acl    models.ACLEntry
	jwtKey []byte
}

func NewDataTokenMiddleware(acl models.ACLEntry, jwtKey []byte) *DataTokenValidMiddleware {
	return &DataTokenValidMiddleware{
		acl:    acl,
		jwtKey: jwtKey,
	}
}

func (dt *DataTokenValidMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("data-access-token")
		if token == "" {
			http.Error(w, "Data access token is missing", http.StatusUnauthorized)
			return
		}

		namespace := mux.Vars(r)["nsid"]
		username, present := r.Context().Value(ctxKeyUsername).(string)
		if !present {
			http.Error(w, "No scope user:name present", http.StatusUnauthorized)
			return
		}

		if err := jwt.ValidateDataAccessToken(token, username, namespace, dt.acl, dt.jwtKey); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
