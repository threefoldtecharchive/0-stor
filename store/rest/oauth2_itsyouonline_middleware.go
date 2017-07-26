package rest

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/manager"
)

type Scope struct {
	Namespace string
	Actor string
	Action string
	Organization string
	Permission string
}

func (s *Scope) Validate() error{
	if s.Namespace == ""||
		s.Action == "" ||
		s.Actor == ""||
		s.Organization == ""||
		s.Permission == ""{
			return errors.New("one or more required fields is empty")
	}

	if s.Permission != "read" &&
		s.Permission != "write" &&
		s.Permission != "delete" &&
		s.Permission != "admin"{
			return errors.New("Invalid permission")
	}

	return nil
}

func (s *Scope) Encode() (string, error){

	if err := s.Validate(); err != nil{
		return "", err
	}


	r := fmt.Sprintf("%s:%s:%s", s.Actor, s.Action, s.Organization)

	if s.Namespace == ""{
		r = fmt.Sprintf("%s.%s", r, "*")
	}else{
		r = fmt.Sprintf("%s.%s", r, s.Namespace)
	}

	if s.Permission != "admin"{
		r = fmt.Sprintf("%s.%s", r, s.Permission)
	}

	return r, nil
}

func (s *Scope) Decode(scope string) error{
	scope = strings.ToLower(scope)

	if strings.Count(scope, ":") != 2{
		return errors.New("Invalid scope string")
	}

	splitted := strings.Split(scope, ":")

	actor := splitted[0]
	action := splitted[1]

	count := strings.Count(splitted[2], ".")

	if count == 0 || count > 2{
		return errors.New("Invalid scope string")
	}

	splitted = strings.Split(splitted[2], ".")

	s.Organization = splitted[0]
	s.Namespace = splitted[1]

	if len(splitted) == 2{
		s.Permission = "admin"
	}else{
		s.Permission = splitted[2]
	}

	s.Action = action
	s.Actor = actor

	if err := s.Validate(); err != nil{
		return err
	}

	return nil
}


// Oauth2itsyouonlineMiddleware is oauth2 middleware for itsyouonline
type Oauth2itsyouonlineMiddleware struct {
	describedBy string
	field       string
	scopes      []*Scope
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

	var s []*Scope

	for _, scopeStr := range scopes{
		scope := new(Scope)
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
func (om *Oauth2itsyouonlineMiddleware) CheckPermissions(r *http.Request, scopes []*Scope) bool {
	for _, s := range scopes{
		for _, scope := range om.scopes{
			if nsid, OK := mux.Vars(r)["nsid"]; OK{
				if scope.Namespace == ""{
					scope.Namespace = nsid
				}
			}

			if s.Namespace == scope.Namespace{
				if s.Permission == "admin" || s.Permission == scope.Permission{
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

		var scopes []*Scope

		if len(oauth2ServerPublicKey) > 0 {
			scopeStrs, err := om.checkJWTGetScope(accessToken)
			if err != nil {
				w.WriteHeader(403)
				return
			}

			for _, scopeStr := range scopeStrs{
				scope := new(Scope)
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
