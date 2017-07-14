package rest

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/config"
	"github.com/zero-os/0-stor/store/core/librairies/reservation"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/rest/models"
)

type DataTokenValidMiddleware struct {
	acl models.ACLEntry
}

func NewDataTokenValidMiddleware(acl models.ACLEntry) *DataTokenValidMiddleware {
	return &DataTokenValidMiddleware{
		acl: acl,
	}
}

func (dt *DataTokenValidMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("data-access-token")
		if token == "" {
			http.Error(w, "Data access token is missing", http.StatusUnauthorized)
			return
		}

		res := models.Reservation{}

		if err := res.ValidateDataAccessToken(dt.acl, token); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type ReservationValidMiddleware struct {
	db     db.DB
	config config.Settings
}

func NewReservationValidMiddleware(db db.DB) *ReservationValidMiddleware {
	return &ReservationValidMiddleware{
		db: db,
		// config: config,
	}
}

func (re *ReservationValidMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		token := r.Header.Get("reservation-token")
		if token == "" {
			http.Error(w, "Reservation token is missing", http.StatusUnauthorized)
			return
		}

		nsid := mux.Vars(r)["nsid"]

		res := models.Reservation{}

		resID, err := res.ValidateReservationToken(token, nsid)

		if err != nil {
			http.Error(w, "Reservation token is invalid", http.StatusUnauthorized)
			return
		}
		res = models.Reservation{
			Namespace: nsid,
			Reservation: reservation.Reservation{
				Id: resID,
			},
		}

		b, err := re.db.Get(res.Key())

		if err != nil {
			if err == db.ErrNotFound {
				http.Error(w, "Reservation token is invalid", http.StatusUnauthorized)
				return

			} else {
				log.Errorln(err.Error())
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}

		if err = res.Decode(b); err != nil {
			log.Errorln(err.Error())
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return

		}
		ctx := context.WithValue(r.Context(), "reservation", res)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Oauth2itsyouonlineMiddleware is oauth2 middleware for itsyouonline
type Oauth2itsyouonlineMiddleware struct {
	describedBy string
	field       string
	scopes      []string
}

var JWTPublicKey *ecdsa.PublicKey

const (
	oauth2ServerPublicKey = `-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAES5X8XrfKdx9gYayFITc89wad4usrk0n2
7MjiGYvqalizeSWTHEpnd7oea9IQ8T5oJjMVH5cc0H5tFSKilFFeh//wngxIyny6
6+Vq5t5B0V0Ehy01+2ceEon2Y0XDkIKv
-----END PUBLIC KEY-----`
)

func init() {
	var err error

	if len(oauth2ServerPublicKey) == 0 {
		return
	}
	JWTPublicKey, err = jwt.ParseECPublicKeyFromPEM([]byte(oauth2ServerPublicKey))
	if err != nil {
		log.Fatalf("failed to parse pub key:%v", err)
	}

}

// NewOauth2itsyouonlineMiddlewarecreate new Oauth2itsyouonlineMiddleware struct
func NewOauth2itsyouonlineMiddleware(scopes []string) *Oauth2itsyouonlineMiddleware {
	om := Oauth2itsyouonlineMiddleware{
		scopes: scopes,
	}

	om.describedBy = "headers"
	om.field = "Authorization"

	return &om
}

// CheckScopes checks whether user has needed scopes
func (om *Oauth2itsyouonlineMiddleware) CheckScopes(scopes []string) bool {
	if len(om.scopes) == 0 {
		return true
	}

	for _, allowed := range om.scopes {
		for _, scope := range scopes {
			if scope == allowed {
				return true
			}
		}
	}
	return false
}

// Handler return HTTP handler representation of this middleware
func (om *Oauth2itsyouonlineMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var accessToken string
		var err error

		// access token checking
		if om.describedBy == "queryParameters" {
			accessToken = r.URL.Query().Get(om.field)
			if accessToken == "" {
				accessToken = r.URL.Query().Get(strings.ToLower(om.field))
			}
		} else if om.describedBy == "headers" {
			accessToken = r.Header.Get(om.field)
			if accessToken == "" {
				accessToken = r.Header.Get(strings.ToLower(om.field))
			}
		}
		if accessToken == "" {
			w.WriteHeader(401)
			return
		}

		var scopes []string
		if len(oauth2ServerPublicKey) > 0 {
			scopes, err = om.checkJWTGetScope(accessToken)
			if err != nil {
				w.WriteHeader(403)
				return
			}
		}

		// check scopes
		if !om.CheckScopes(scopes) {
			log.Debug("no required scopes")
			w.WriteHeader(403)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// check JWT token and get it's scopes
func (om *Oauth2itsyouonlineMiddleware) checkJWTGetScope(tokenStr string) ([]string, error) {
	jwtStr := strings.TrimSpace(strings.TrimPrefix(tokenStr, "Bearer"))
	token, err := jwt.Parse(jwtStr, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodES384 {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return JWTPublicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return nil, fmt.Errorf("invalid token")
	}

	var scopes []string
	for _, v := range claims["scope"].([]interface{}) {
		scopes = append(scopes, v.(string))
	}
	return scopes, nil
}
