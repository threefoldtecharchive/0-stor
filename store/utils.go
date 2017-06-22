package main

import (
	"math"
	"encoding/binary"
	"crypto/rand"
	"encoding/base64"
)

func Float64frombytes(bytes []byte) float64 {
	bits := binary.LittleEndian.Uint64(bytes)
	float := math.Float64frombits(bits)
	return float
}

func Float64bytes(float float64) []byte {
	bits := math.Float64bits(float)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	return bytes
}

/* Random bytes generator */
func GenerateRandomBytes(n int) ([]byte, error){
	b := make([]byte, n)

	_, err := rand.Read(b)

	if err != nil {
		return nil, err
	}
	return b, nil
}

func GenerateUUID(n int) (string, error){
	b, err := GenerateRandomBytes(n)

	token, err := base64.URLEncoding.EncodeToString(b), err
	if err != nil{
		return "", err
	}
	return token, nil

}

func InvalidateToken(token string) error{
	//@TODO: Invalidate GWT token
	return nil
}

