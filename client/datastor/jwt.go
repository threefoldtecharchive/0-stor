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

package datastor

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/zero-os/0-stor/client/itsyouonline"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/hashicorp/golang-lru/simplelru"
)

// JWTTokenGetter defines the interface of a type which can provide us
// with a valid JWT token at all times,
// as well as the option to get a label given a namespace,
// and claims given a valid JWT Token.
type JWTTokenGetter interface {
	// GetJWTToken gets a cached JWT token or create a new JWT token should it be invalid.
	//
	// The implementation can return an error,
	// should it not be possible to return a valid JWT Token for whatever reason,
	// if no error is returned, it should be assumed that the returned value is valid.
	GetJWTToken(namespace string) (string, error)

	// GetLabel gets the label for the given namespace
	GetLabel(namespace string) (string, error)

	// GetClaimsFromJWTToken returns the claims from a JWTToken,
	// retrieved using the GetJWTToken method from the same type as this method.
	GetClaimsFromJWTToken(token string) (map[string]interface{}, error)
}

// IYOClient defines the minimal, and only, interface we require,
// in order to create JWT tokens using the IYO Web API.
type IYOClient interface {
	CreateJWT(namespace string, perms itsyouonline.Permission) (string, error)
}

// JWTTokenGetterUsingIYOClient creates a JWTTokenGetter,
// used to create JWT tokens using the IYO Web API.
func JWTTokenGetterUsingIYOClient(organization string, client IYOClient) (*IYOBasedJWTTokenGetter, error) {
	if len(organization) == 0 {
		return nil, errors.New("no organization given")
	}
	if client == nil {
		return nil, errors.New("no IYO client given")
	}
	return &IYOBasedJWTTokenGetter{
		prefix: organization + "_0stor_",
		client: client,
	}, nil
}

// IYOBasedJWTTokenGetter is a simpler wrapper which we define for our itsyouonline client,
// as to provide a JWT Token Getter, using the IYO client.
type IYOBasedJWTTokenGetter struct {
	prefix string
	client IYOClient
}

// GetJWTToken implements JWTTokenGetter.GetJWTToken
func (iyo *IYOBasedJWTTokenGetter) GetJWTToken(namespace string) (string, error) {
	return iyo.client.CreateJWT(
		namespace,
		itsyouonline.Permission{
			Read:   true,
			Write:  true,
			Delete: true,
			Admin:  true,
		})
}

// GetLabel implements JWTTokenGetter.GetLabel
func (iyo *IYOBasedJWTTokenGetter) GetLabel(namespace string) (string, error) {
	if namespace == "" {
		return "", errors.New("iyoJWTTokenGetter: no/empty namespace given")
	}
	return iyo.prefix + namespace, nil
}

// GetClaimsFromJWTToken implements JWTTokenGetter.GetClaimsFromJWTToken
func (iyo *IYOBasedJWTTokenGetter) GetClaimsFromJWTToken(tokenStr string) (map[string]interface{}, error) {
	jwtStr := strings.TrimSpace(strings.TrimPrefix(tokenStr, "Bearer"))
	token, err := jwtgo.Parse(jwtStr, func(token *jwtgo.Token) (interface{}, error) {
		if token.Method != jwtgo.SigningMethodES384 {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return iyoPublicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwtgo.MapClaims)
	if !(ok && token.Valid) {
		return nil, errors.New("invalid JWT token")
	}
	return claims, nil
}

// CachedJWTTokenGetter turns any valid (non-nil and hopefully uncached) JWTTokenGetter,
// into a cached JWTTokenGetter.
//
// If cacheSize is 0 or less, `DefaultJWTCacheBucketSize` is used as the bucket size.
// If bucketCount is 0 or less, `DefaultJWTCacheBucketCount` is used as the bucket count.
//
// storage is optional and can be nil, in which case nothing is stored.
//
// See JWTTokenGetterCache for more information.
func CachedJWTTokenGetter(getter JWTTokenGetter, bucketCount, bucketSize int) (*JWTTokenGetterCache, error) {
	if getter == nil {
		return nil, errors.New("CachedJWTTokenGetter: no JWTTokenGetter given")
	}
	if bucketCount <= 0 {
		bucketCount = DefaultJWTCacheBucketCount
	} else if bucketCount > math.MaxUint32 {
		return nil, errors.New("CachedJWTTokenGetter: invalid bucket count")
	}
	if bucketSize <= 0 {
		bucketSize = DefaultJWTCacheBucketSize
	}

	// create cache and its internal buckets
	cache := &JWTTokenGetterCache{
		JWTTokenGetter: getter,
		buckets:        make([]*jwtTokenCacheBucket, bucketCount),
		bucketCount:    uint32(bucketCount),
	}
	for i := range cache.buckets {
		cache.buckets[i] = newJWTTokenCacheBucket(getter, bucketSize)
	}

	// return our fresh-created cache, ready for usage
	return cache, nil
}

// JWTTokenGetterCache is a JWTTokenGetter which can be used
// to turn any valid (and hopefully uncached) JWTTokenGetter, into
// a cached JWTTokenGetter.
type JWTTokenGetterCache struct {
	JWTTokenGetter
	buckets     []*jwtTokenCacheBucket
	bucketCount uint32
}

// GetJWTToken implements JWTTokenGetter.GetJWTToken
func (c *JWTTokenGetterCache) GetJWTToken(namespace string) (string, error) {
	// TODO: research if there is a faster algorithm that we might
	// want to use instead of the std ChecksumIEEE,
	// for now this one is used, inspired by golang's semi-standard memcache.
	index := crc32.ChecksumIEEE([]byte(namespace)) % c.bucketCount
	return c.buckets[index].GetJWTToken(namespace)
}

// newJWTTokenCacheBucket creates a new JWTTokenCacheBucket,
// using any valid JWTTokenGetter as its creation platform.
func newJWTTokenCacheBucket(getter JWTTokenGetter, size int) *jwtTokenCacheBucket {
	cache, err := simplelru.NewLRU(size, nil)
	if err != nil {
		// NewLRU only returns an error when the size is 0 or negative,
		// as we control and validate the size, this cannot happen
		panic("unexpected LRU creation error: " + err.Error())
	}
	return &jwtTokenCacheBucket{
		getter: getter,
		cache:  cache,
	}
}

// jwtTokenCacheBucket defines a single bucket of a bigger JWT Token cache,
// we use buckets as a strategy to help mitigate the cost of locking.
type jwtTokenCacheBucket struct {
	getter   JWTTokenGetter
	cache    *simplelru.LRU
	cacheMux sync.Mutex
}

// GetJWTToken implements JWTTokenGetter.GetJWTToken
func (b *jwtTokenCacheBucket) GetJWTToken(namespace string) (string, error) {
	// The reason we choose to lock over the entire scope of this function,
	// is such that we want to prevent that we hit the IYO webserver multiple times for the same token,
	// as can be happen due to the async nature of our storage code,
	// meaning that from multiple goroutines at once, for the same namespace (and even data),
	// we might want to write to a zstordb server.
	// If instead we would have just used a type that locks locally as part of `cache.Get`,
	// those multiple goroutines would all think the token is not yet in the cache,
	// and will all try to add it, and thus we would get multiple hits to the IYO webserver.
	// Not sure if this is the best lock/cache strategy to go for,
	// but hopefully this clarifies why we currently lock at such a high level.
	b.cacheMux.Lock()
	defer b.cacheMux.Unlock()

	// check if a token is cached for the given namespace,
	// and if so, that the token still has a valid TTL
	if v, ok := b.cache.Get(namespace); ok {
		token := v.(*cachedJWTToken)
		if time.Until(token.ExpirationTime).Seconds() >= jwtTokenCacheMinTTLInSeconds {
			// return the valid cached token
			return token.Token, nil
		}
		// token was cached for this namespace,
		// but is invalid by now
		b.cache.Remove(namespace)
	}

	// create token using the internal/wrapped JWTTokenGetter
	token, err := b.getter.GetJWTToken(namespace)
	if err != nil {
		return "", err
	}

	// validate token, get the expiration time and ensure
	// the token's TTL is long enough
	exp, err := b.getExpirationTimeFromJWTToken(token)
	if err != nil {
		return "", err
	}
	if time.Until(exp).Seconds() < jwtTokenCacheMinTTLInSeconds {
		return "", errors.New("retrieved about-to-be expired token from JWTTokenGetter")
	}

	// cache the token for future usage
	b.cache.Add(namespace, &cachedJWTToken{
		Token:          token,
		ExpirationTime: exp,
	})

	// return the token, by value, for consumption
	return token, nil
}

// getExpirationTimeFromJWTToken is a utility function to extract the expiration time,
// from a token string, retrieving it from the claims of that parsed token string.
func (b *jwtTokenCacheBucket) getExpirationTimeFromJWTToken(token string) (time.Time, error) {
	claims, err := b.getter.GetClaimsFromJWTToken(token)
	if err != nil {
		return time.Time{}, err
	}
	exp, ok := claims["exp"].(float64)
	if !ok {
		return time.Time{}, errors.New("invalid expiration claim in token")
	}
	return time.Unix(int64(exp), 0), nil
}

// cachedJWTToken is a utility type,
// used to cache a Token (string) together with its expiration time,
// such that we only have to extract and parse the expiration time once.
type cachedJWTToken struct {
	Token          string
	ExpirationTime time.Time
}

const (
	// DefaultJWTCacheBucketCount is the default bucket count used,
	// as the bucket count of a JWTTokenGetterCache
	DefaultJWTCacheBucketCount = 16
	// DefaultJWTCacheBucketSize is the default cache size used
	// as the size of the LRU cache, of a single bucket,
	// used internally of a JWTTokenGetterCache.
	DefaultJWTCacheBucketSize = 256
)

var (
	// jwtTokenCacheMinTTLInSeconds defines the minimum amount of seconds
	// a cached or created JWT Token is expected to have,
	// in order to not be marked as invalid.
	jwtTokenCacheMinTTLInSeconds float64 = 30
)

var (
	// IYO's (https://itsyou.online/) JWT signature public key
	iyoPublicKey = func() *ecdsa.PublicKey {
		pubKey, err := jwtgo.ParseECPublicKeyFromPEM([]byte(`-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAES5X8XrfKdx9gYayFITc89wad4usrk0n2
7MjiGYvqalizeSWTHEpnd7oea9IQ8T5oJjMVH5cc0H5tFSKilFFeh//wngxIyny6
6+Vq5t5B0V0Ehy01+2ceEon2Y0XDkIKv
-----END PUBLIC KEY-----`))
		if err != nil {
			panic(fmt.Sprintf("failed to parse public IYO key: %v", err))
		}
		return pubKey
	}()
)

var (
	_ JWTTokenGetter = (*IYOBasedJWTTokenGetter)(nil)
	_ JWTTokenGetter = (*JWTTokenGetterCache)(nil)
)
