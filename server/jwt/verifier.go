/*
 * Copyright (C) 2017-2018 GIG Technology NV and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jwt

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/karlseguin/ccache"
	"google.golang.org/grpc/metadata"
)

const (
	// IYO's(https://itsyou.online/) JWT signature public key
	iyoPublicKeyStr = `-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAES5X8XrfKdx9gYayFITc89wad4usrk0n2
7MjiGYvqalizeSWTHEpnd7oea9IQ8T5oJjMVH5cc0H5tFSKilFFeh//wngxIyny6
6+Vq5t5B0V0Ehy01+2ceEon2Y0XDkIKv
-----END PUBLIC KEY-----`

	// number of entries we want to keep in the LRU cache
	// rough estimation size of jwtCacheVal is 150 bytes
	// if our jwtCacheSize = 1024, it takes : 150 * 1024 bytes = 153 kilobytes
	jwtCacheSize = 2 << 11 // 4096
)

var (
	defVerifier           *Verifier
	createDefVerifierOnce sync.Once
)

// Error variables
var (
	// ErrNoJWTToken is return when the grpc request doesn't provide a valid jwt token
	ErrNoJWTToken = errors.New("No jwt token in context")

	// ErrWrongScopes is returned when a JWT doesn't contain the required scopes
	ErrWrongScopes = errors.New("JWT token doesn't contains required scopes")
)

// TokenVerifier represents a JWT token verifier interface,
// used and designed for the 0-stor server.
type TokenVerifier interface {
	// ValidateJWT validates a JWT for 0-stor
	ValidateJWT(ctx context.Context, method Method, label string) error
}

// NopVerifier implements TokenVerifier,
// and returns always nil.
// It is to be used there where you do not want/need a verifier.
type NopVerifier struct{}

// ValidateJWT implements TokenVerifier.ValidateJWT
func (nv NopVerifier) ValidateJWT(context.Context, Method, string) error { return nil }

// DefaultVerifier returns a JWT verifier
// with the IYO(https://itsyou.online/) public key.
func DefaultVerifier() *Verifier {
	createDefVerifierOnce.Do(func() {
		var err error
		defVerifier, err = NewVerifier(iyoPublicKeyStr)
		if err != nil {
			panic(err)
		}
	})

	return defVerifier
}

// NewVerifier returns a JWT verifier with provided public key.
// The public key should be PEM-encoded.
func NewVerifier(pubKeyStr string) (*Verifier, error) {
	pubKey, err := jwtgo.ParseECPublicKeyFromPEM([]byte(pubKeyStr))
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	conf := ccache.Configure()
	conf.MaxSize(jwtCacheSize)
	jwtCache := ccache.New(conf)

	return &Verifier{
		key:   pubKey,
		cache: jwtCache,
	}, nil
}

// Verifier represents a JWT verifier for the 0-stor
type Verifier struct {
	key   crypto.PublicKey
	cache *ccache.Cache
}

// ValidateJWT implements the Verifier.ValidateJWT interface
func (v *Verifier) ValidateJWT(ctx context.Context, method Method, label string) error {
	var scopes []string
	token, err := v.extractJWTToken(ctx)
	if err != nil {
		return err
	}

	scopes, err = v.getScopes(token)
	if err != nil {
		return err
	}

	expected, err := v.expectedScopes(method, label)
	if err != nil {
		return err
	}

	if !v.checkPermissions(expected, scopes) {
		return ErrWrongScopes
	}

	return nil
}

// GetScopes returns the scopes from a JWT
func (v *Verifier) getScopes(token string) ([]string, error) {
	scopes, err := v.getScopesFromCache(token)
	if err != nil {
		// there is error in token
		return nil, err
	}

	// check if scopes are returned from cache
	if scopes != nil {
		return scopes, nil
	}

	scopes, exp, err := v.checkJWTGetScopes(token)

	if err != nil || time.Until(time.Unix(exp, 0)).Seconds() < 0 {
		// Insert invalid or expired token to cache
		// so we don't need to validate it again
		v.cache.Set(token, jwtCacheVal{
			valid: false,
		}, time.Hour*24)

		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("expired token")
	}

	// insert valid token to cache
	cacheVal := jwtCacheVal{
		valid:  true,
		scopes: scopes,
	}
	v.cache.Set(token, cacheVal, time.Until(time.Unix(exp, 0)))
	return scopes, nil
}

// ExtractJWTToken extracts the JWT token
func (v *Verifier) extractJWTToken(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", ErrNoJWTToken
	}

	token, ok := md[rpctypes.MetaAuthKey]
	if !ok || len(token) < 1 {
		return "", ErrNoJWTToken
	}

	return token[0], nil
}

// ExpectedScopes deducts the required scope based on the request method
func (v *Verifier) expectedScopes(method Method, label string) ([]string, error) {
	if err := v.validateNamespaceLabel(label); err != nil {
		return nil, err
	}

	adminScope := strings.Replace(label, "_0stor_", ".0stor.", 1)

	if method == MethodAdmin {
		// return admin scope
		// e.g.: first.0stor.gig
		return []string{adminScope}, nil
	}

	return []string{
		// add scope the requested method
		// e.g.: first.0stor.gig.read
		fmt.Sprintf("%s.%s", adminScope, method),
		// add admin scope
		// e.g.: first.0stor.gig
		adminScope,
	}, nil
}

// CheckPermissions checks if the user has the right 0-stor scopes
func (v *Verifier) checkPermissions(expectedScopes, userScopes []string) bool {
	for _, scope := range userScopes {
		scope = strings.Replace(scope, "user:memberof:", "", 1)
		for _, expected := range expectedScopes {
			if scope == expected {
				return true
			}
		}
	}

	return false
}

// ValidateNamespaceLabel checks if a label follows the 0-stor namespace pattern convention
func (v *Verifier) validateNamespaceLabel(nsid string) error {
	// subOrg_0stor_org i.e first_0stor_gig
	if strings.Count(nsid, "_0stor_") != 1 || strings.HasSuffix(nsid, "_0stor_") {
		return fmt.Errorf("Invalid namespace label: %s", nsid)
	}

	return nil
}

// CheckJWTGetScopes gets and checks the JWT and returns the scopes and expiration
func (v *Verifier) checkJWTGetScopes(tokenStr string) ([]string, int64, error) {
	jwtStr := strings.TrimSpace(strings.TrimPrefix(tokenStr, "Bearer"))
	var scopes []string
	var exp int64

	token, err := jwtgo.Parse(jwtStr, func(token *jwtgo.Token) (interface{}, error) {
		if token.Method != jwtgo.SigningMethodES384 {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return v.key, nil
	})
	if err != nil {
		return scopes, exp, err
	}

	claims, ok := token.Claims.(jwtgo.MapClaims)
	if !(ok && token.Valid) {
		return scopes, exp, fmt.Errorf("invalid token")
	}

	for _, v := range claims["scope"].([]interface{}) {
		scopes = append(scopes, v.(string))
	}

	expFl, ok := claims["exp"].(float64)
	if !ok {
		return scopes, exp, fmt.Errorf("invalid expiration claims in token")
	}
	exp = int64(expFl)
	return scopes, exp, nil
}

func (v *Verifier) getScopesFromCache(token string) ([]string, error) {
	item := v.cache.Get(token)
	if item == nil {
		return nil, nil
	}

	// check validity
	cacheVal := item.Value().(jwtCacheVal)
	if !cacheVal.valid {
		return nil, fmt.Errorf("invalid token")
	}

	// check cache expiration
	if item.Expired() {
		v.cache.Delete(token)
		return nil, fmt.Errorf("expired token")
	}

	return cacheVal.scopes, nil
}

// jwtCacheVal represents the value stored in the cache
type jwtCacheVal struct {
	valid  bool
	scopes []string
}
