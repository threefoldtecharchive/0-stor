package rest

import (
	"crypto/ecdsa"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"net/http"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/zero-os/0-stor/server/db"
	"github.com/zero-os/0-stor/server/manager"
	"github.com/pkg/errors"
)


// Oauth2itsyouonlineMiddleware is oauth2 middleware for itsyouonline
type Oauth2itsyouonlineMiddleware struct {
	describedBy string
	field       string
	pubKey      *ecdsa.PublicKey
	url         string
	db          db.DB
}

const (
	oauth2ServerPublicKey = `-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAES5X8XrfKdx9gYayFITc89wad4usrk0n2
7MjiGYvqalizeSWTHEpnd7oea9IQ8T5oJjMVH5cc0H5tFSKilFFeh//wngxIyny6
6+Vq5t5B0V0Ehy01+2ceEon2Y0XDkIKv
-----END PUBLIC KEY-----` // fill it with oauth2 server public key

	oauth2ServerUrl = `https://itsyou.online/`
)

// NewOauth2itsyouonlineMiddlewarecreate new Oauth2itsyouonlineMiddleware struct
func NewOauth2itsyouonlineMiddleware(db db.DB) *Oauth2itsyouonlineMiddleware {
	om := Oauth2itsyouonlineMiddleware{
		describedBy:"headers",
		field: "Authorization",
		url: oauth2ServerUrl,
		db: db,
	}

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
func (om *Oauth2itsyouonlineMiddleware) CheckPermissions(expectedScopes []string, userScopes []string) bool {
	for _, scope := range userScopes{
		scope = strings.Replace(scope, "user:memberof:", "", 1)
		for _, expected := range expectedScopes{
			if scope == expected{
				return true
			}
		}
	}
	return false
}

func (om *Oauth2itsyouonlineMiddleware) ValidateNsid(nsid string) error{

	// subOrg_0stor_org i.e first_0stor_gig
	if strings.Count(nsid, "_0stor_") != 1 ||  strings.HasSuffix(nsid, "_0stor_"){
		err := fmt.Sprintf("Invalid nsid %s", nsid)
		log.Info(err)
		return errors.New(err)
	}

	return nil
}

func (om *Oauth2itsyouonlineMiddleware) GetExpectedScopes(r *http.Request, nsid string) ([]string, error){
	permissions := map[string]string{
		"GET": "read",
		"POST": "write",
		"DELETE": "delete",
		"PUT" : "write",
		"HEAD": "read",
	}

	perm, OK := permissions[r.Method]

	if !OK {
		return []string{}, errors.New(fmt.Sprintf("No permission specified for HTTP method %v", r.Method))
	}

	adminScope := strings.Replace(nsid, "_0stor_", ".0stor.", 1)

	return []string{
		// example :: first.0stor.gig.read
		fmt.Sprintf("%s.%s", adminScope, perm),
		//admin ::first.0stor.gig
		adminScope,
	}, nil
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

		// nsid checking
		var nsid string

		nsid, OK := mux.Vars(r)["nsid"]

		if !OK{
			w.WriteHeader(400)
			return
		}

		if err := om.ValidateNsid(nsid); err != nil{
			log.Errorln(err.Error())
			w.WriteHeader(400)
			return
		}

		expectedScope, err := om.GetExpectedScopes(r, nsid)

		if err != nil{
			log.Errorln(err.Error())
			w.WriteHeader(400)
			return
		}

		log.Infof("[IYO] Expected scope : %v", expectedScope)

		if len(oauth2ServerPublicKey) > 0 {
			userScopes, err := om.checkJWTGetScope(accessToken)
			log.Infof("[IYO] User scopes : %v", userScopes)

			if err != nil {
				log.Infof(err.Error())
				w.WriteHeader(403)
				return
			}

			// check scopes
			if !om.CheckPermissions(expectedScope, userScopes) {

				w.WriteHeader(403)
				return
			}

			// Create namespace if not exists

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
