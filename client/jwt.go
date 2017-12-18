package client

import (
	"errors"

	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/client/itsyouonline"
)

type iyoClient interface {
	CreateJWT(namespace string, perms itsyouonline.Permission) (string, error)
}

func jwtTokenGetterFromIYOClient(organization string, client iyoClient) *iyoJWTTokenGetter {
	if len(organization) == 0 {
		panic("no organization given")
	}
	if client == nil {
		panic("no IYO client given")
	}
	return &iyoJWTTokenGetter{
		prefix: organization + "_0stor_",
		client: client,
	}
}

// iyoJWTTokenGetter is a simpler wrapper which we define for our itsyouonline client,
// as to provide a JWT Token Getter, using the IYO client.
type iyoJWTTokenGetter struct {
	prefix string
	client iyoClient
}

// GetJWTToken implements datastor.JWTTokenGetter.GetJWTToken
func (iyo *iyoJWTTokenGetter) GetJWTToken(namespace string) (string, error) {
	return iyo.client.CreateJWT(
		namespace,
		itsyouonline.Permission{
			Read:   true,
			Write:  true,
			Delete: true,
			Admin:  true,
		})
}

// GetLabel implements datastor.JWTTokenGetter.GetLabel
func (iyo *iyoJWTTokenGetter) GetLabel(namespace string) (string, error) {
	if namespace == "" {
		return "", errors.New("iyoJWTTokenGetter: no/empty namespace given")
	}
	return iyo.prefix + namespace, nil
}

var (
	_ datastor.JWTTokenGetter = (*iyoJWTTokenGetter)(nil)
)
