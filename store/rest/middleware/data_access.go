package midleware

import (
	"net/http"

	"github.com/zero-os/0-stor/store/jwt"
	"github.com/zero-os/0-stor/store/rest/models"
)

type DataTokenValidMiddleware struct {
	acl    models.ACLEntry
	jwtKey []byte
}

func NewDataTokenValidMiddleware(acl models.ACLEntry, jwtKey []byte) *DataTokenValidMiddleware {
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

		if err := jwt.ValidateDataAccessToken(dt.acl, token); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
