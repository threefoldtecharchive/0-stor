package jwt

import (
	"errors"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/zero-os/0-stor/store/rest/models"
)

var (
	ErrWrongAlg       = errors.New("reservation token not valid, wrong singning algorithm")
	ErrWrongNamespace = errors.New("reservation token not valid, wrong namespace")
	ErrWrongIssuer    = errors.New("reservation token not valid, wrong issuer")
)

type reservationClaims struct {
	jwt.StandardClaims
	ID           string // ID of the reservation
	AdminID      string // ID of the user that has created the reservation, only him can renew
	SizeReserved uint64
	Namespace    string
}

func GenerateReservationToken(reservation models.Reservation, key []byte) (string, error) {

	claims := reservationClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: (time.Time)(reservation.ExpireAt).Unix(),
			IssuedAt:  (time.Time)(reservation.Created).Unix(),
			Issuer:    "0-stor", //TODO uniq id for the 0-stor
		},
		SizeReserved: reservation.SizeReserved,
		AdminID:      reservation.AdminId,
		ID:           reservation.Id,
		Namespace:    reservation.Namespace,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(key)
}

func ValidateReservationToken(tokenString, namespace string, key []byte) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &reservationClaims{}, func(token *jwt.Token) (interface{}, error) {
		return key, nil
	})

	if alg, present := token.Header["alg"]; !present || alg != jwt.SigningMethodHS256.Alg() {
		return "", ErrWrongAlg
	}

	if claims, ok := token.Claims.(*reservationClaims); ok && token.Valid {
		if claims.Namespace != namespace {
			return "", ErrWrongNamespace
		}
		if claims.Issuer != "0-stor" {
			return "", ErrWrongIssuer
		}
		return claims.ID, nil
	}

	return "", err
}
