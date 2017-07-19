package middleware

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/dgrijalva/jwt-go"
)

var iyoKey = []byte(`-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAES5X8XrfKdx9gYayFITc89wad4usrk0n2
7MjiGYvqalizeSWTHEpnd7oea9IQ8T5oJjMVH5cc0H5tFSKilFFeh//wngxIyny6
6+Vq5t5B0V0Ehy01+2ceEon2Y0XDkIKv
-----END PUBLIC KEY-----`)

// Oauth2itsyouonlineMiddleware is oauth2 middleware for itsyouonline
type Oauth2itsyouonlineMiddleware struct {
	describedBy string
	field       string
	scopes      []string
	key         *ecdsa.PublicKey
}

const ctxKeyUsername = "0stor:username"

// NewOauth2itsyouonlineMiddlewarecreate new Oauth2itsyouonlineMiddleware struct
func NewOauth2itsyouonlineMiddleware(scopes []string) *Oauth2itsyouonlineMiddleware {
	pubKey, err := jwt.ParseECPublicKeyFromPEM(iyoKey)
	if err != nil {
		log.Fatalf("failed to parse pub key:%v", err)
	}

	om := Oauth2itsyouonlineMiddleware{
		scopes: scopes,
		key:    pubKey,
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

		claims, err := om.checkJWTGetScope(accessToken)
		if err != nil {
			log.Errorln(err)
			w.WriteHeader(403)
			return
		}

		// extract scope from claims
		var scopes []string
		for _, v := range claims["scope"].([]interface{}) {
			scopes = append(scopes, v.(string))
		}

		// check scopes
		if !om.CheckScopes(scopes) {
			log.Errorln("no required scopes")
			w.WriteHeader(403)
			return
		}

		// pass the username in the context of the requests
		username, present := claims["username"]
		if present {
			ctx := context.WithValue(r.Context(), ctxKeyUsername, username)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			next.ServeHTTP(w, r)
		}

	})
}

// check JWT token and returns the claims contain in the token
func (om *Oauth2itsyouonlineMiddleware) checkJWTGetScope(tokenStr string) (jwt.MapClaims, error) {
	jwtStr := strings.TrimSpace(strings.TrimPrefix(tokenStr, "Bearer"))
	token, err := jwt.Parse(jwtStr, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodES384 {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return om.key, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !(ok && token.Valid) {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
