package jwt

import (
	"errors"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/zero-os/0-stor/store/rest/models"
)

var (
	ErrTokenInvalid = errors.New("data access token not valid")
	ErrWrongUser    = errors.New("data access token not valid, wrong user")
	ErrWrongACL     = errors.New("data access token not valid, wrong permission")
)

type dataAccessClaims struct {
	jwt.StandardClaims
	Namespace string
	User      string
	ACL       models.ACLEntry
}

func GenerateDataAccessToken(user string, reservation models.Reservation, acl models.ACLEntry, key []byte) (string, error) {
	claims := dataAccessClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: (time.Time)(reservation.ExpireAt).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "0-stor", //TODO uniq id for the 0-stor
		},
		Namespace: reservation.Namespace,
		User:      user,
		ACL:       acl,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

func ValidateDataAccessToken(tokenString, user, namespace string, acl models.ACLEntry, key []byte) error {
	token, err := jwt.ParseWithClaims(tokenString, &dataAccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})

	if alg, present := token.Header["alg"]; !present || alg != jwt.SigningMethodHS256.Alg() {
		return ErrWrongAlg
	}

	if claims, ok := token.Claims.(*dataAccessClaims); ok && token.Valid {
		if claims.Namespace != namespace {
			return ErrWrongNamespace
		}
		if claims.Issuer != "0-stor" {
			return ErrWrongIssuer
		}
		if claims.User != user {
			return ErrWrongUser
		}

		if claims.ACL.Admin {
			// Admin has all right
			return nil
		}
		// HTTP action ACL requires missing permission granted for that user
		if (acl.Admin && !claims.ACL.Admin) ||
			(acl.Read && !claims.ACL.Read) ||
			(acl.Write && !claims.ACL.Write) ||
			(acl.Delete && !claims.ACL.Delete) {
			return ErrWrongACL
		}
		return nil
	}

	return err
}
