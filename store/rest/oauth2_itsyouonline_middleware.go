package rest

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/manager"
	"github.com/zero-os/0-stor/store/scope"
)


// Oauth2itsyouonlineMiddleware is oauth2 middleware for itsyouonline
type Oauth2itsyouonlineMiddleware struct {
	describedBy string
	field       string
	scopes      []*scope.Scope
	pubKey      *ecdsa.PublicKey
	url         string
	db          db.DB
}

const (
	oauth2ServerPublicKey = `-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEkmd07vxBqoCiHsaplIpjlonDeOnpvPam
ORMdBcAlHNXbzwplcdK4qlZGPBz9mxDSrBOv9SZH+Et6r8gn9Fx/+ZjlvRwowqOU
FpCIijAEx6A3BhfRUbmwl1evBKzWB/qw
-----END PUBLIC KEY-----` // fill it with oauth2 server public key
	oauth2ServerUrl = `https://staging.itsyou.online`
)

// NewOauth2itsyouonlineMiddlewarecreate new Oauth2itsyouonlineMiddleware struct
func NewOauth2itsyouonlineMiddleware(db db.DB, scopes []string) *Oauth2itsyouonlineMiddleware {

	var s []*scope.Scope

	for _, scopeStr := range scopes{
		scope := new(scope.Scope)
		if err := scope.Decode(scopeStr); err != nil{
			log.Fatal("Invalid scope")
		}
		s = append(s, scope)
	}

	om := Oauth2itsyouonlineMiddleware{
		scopes: s,
	}

	om.describedBy = "headers"
	om.field = "Authorization"
	om.url = oauth2ServerUrl

	if len(oauth2ServerPublicKey) > 0 {
		JWTPublicKey, err := jwt.ParseECPublicKeyFromPEM([]byte(oauth2ServerPublicKey))
		if err != nil {
			log.Fatalf("failed to parse pub key:%v", err)
		}
		om.pubKey = JWTPublicKey
	}
	return &om
}

// CheckScopes checks whether user has needed scopes
func (om *Oauth2itsyouonlineMiddleware) CheckPermissions(r *http.Request, scopes []*scope.Scope) bool {
	for _, actual := range scopes{
		for _, expected := range om.scopes{
			if nsid, OK := mux.Vars(r)["nsid"]; OK{
				if expected.Namespace == ""{
					expected.Namespace = nsid
				}
			}

			if actual.Namespace == expected.Namespace &&
				actual.Organization == expected.Organization&&
				actual.Actor == expected.Actor && actual.Action == expected.Action{
					if actual.Permission == "admin" || actual.Permission == expected.Permission{
						return true
					}
			}
		}
	}

	return false
}


// Handler return HTTP handler representation of this middleware
func (om *Oauth2itsyouonlineMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var accessToken string

		// access token checking
		if om.describedBy == "queryParameters" {
			accessToken = r.URL.Query().Get(om.field)
		} else if om.describedBy == "headers" {
			accessToken = r.Header.Get(om.field)
		}
		if accessToken == "" {
			w.WriteHeader(401)
			return
		}

		var scopes []*scope.Scope

		if len(oauth2ServerPublicKey) > 0 {
			scopeStrs, err := om.checkJWTGetScope(accessToken)
			if err != nil {
				w.WriteHeader(403)
				return
			}

			for _, scopeStr := range scopeStrs{
				scope := new(scope.Scope)
				if err := scope.Decode(scopeStr); err != nil{
					w.WriteHeader(403)
					return
				}

				// If namespace not defined ; i.e == "", replace with current
				if nsid, OK := mux.Vars(r)["nsid"]; OK{
					if scope.Namespace == ""{
						scope.Namespace = nsid
					}
				}
				scopes = append(scopes, scope)
			}
		}

		// check scopes
		if !om.CheckPermissions(r, scopes) {
			w.WriteHeader(403)
			return
		}

		// Create namespace if not exists
		if nsid, OK := mux.Vars(r)["nsid"]; OK{
			if err := manager.NewNamespaceManager(om.db).Create(nsid); err != nil{
				w.WriteHeader(500)
				return
			}
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
		return om.pubKey, nil
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
