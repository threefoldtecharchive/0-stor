package client

import (
	"github.com/zero-os/0-stor/client/datastor"
	"github.com/zero-os/0-stor/client/itsyouonline"
)

// iyoJWTTokenGetter is a simpler wrapper which we define for our itsyouonline client,
// as to provide a JWT Token Getter, using the IYO client.
type iyoJWTTokenGetter struct {
	client interface {
		CreateJWT(namespace string, perms itsyouonline.Permission) (string, error)
	}
}

// GetJWTToken implements datastor.JWTTokenGetter.GetJWTToken
func (iyo *iyoJWTTokenGetter) GetJWTToken(namespace string) (string, error) {
	return iyo.client.CreateJWT(
		namespace,
		itsyouonline.Permission{
			Read:   true,
			Write:  true,
			Delete: true,
		})
}

var (
	_ datastor.JWTTokenGetter = (*iyoJWTTokenGetter)(nil)
)
