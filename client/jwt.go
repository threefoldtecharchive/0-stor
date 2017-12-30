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
