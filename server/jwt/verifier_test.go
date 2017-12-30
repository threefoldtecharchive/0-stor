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
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	iyo "github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/server/api/grpc/rpctypes"

	jwtgo "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

const (
	testPubKeyPath  = "../../devcert/jwt_pub.pem"
	testPrivKeyPath = "../../devcert/jwt_key.pem"
)

func TestGetNopVerifier(t *testing.T) {
	verifier := NopVerifier{}
	require.Nil(t, verifier.ValidateJWT(nil, MethodAdmin, ""))
}

func getTestVerifier(authEnabled bool) (TokenVerifier, error) {
	if !authEnabled {
		return NopVerifier{}, nil
	}

	pubKey, err := ioutil.ReadFile(testPubKeyPath)
	if err != nil {
		return nil, err
	}
	return NewVerifier(string(pubKey))
}

func TestValidateJWT(t *testing.T) {
	require := require.New(t)

	org := "org"
	namespace := "ns"
	label := fmt.Sprintf("%s_0stor_%s", org, namespace)

	verifier, err := getTestVerifier(true)
	require.NoError(err, "failed to create test verifier")

	adminPerm := iyo.Permission{Admin: true}
	readPerm := iyo.Permission{Read: true}
	writePerm := iyo.Permission{Write: true}
	delPerm := iyo.Permission{Delete: true}
	allButAdminPerm := iyo.Permission{
		Read:   true,
		Write:  true,
		Delete: true,
	}

	adminToken := getToken(require, adminPerm, org, namespace)
	readToken := getToken(require, readPerm, org, namespace)
	writeToken := getToken(require, writePerm, org, namespace)
	delToken := getToken(require, delPerm, org, namespace)
	allButAdminToken := getToken(require, allButAdminPerm, org, namespace)

	// test valid cases

	validCases := []validateJWTCase{
		// check if admin token has rights on every method
		{
			tokenStr:   adminToken,
			permission: adminPerm,
			method:     MethodAdmin,
			label:      label,
		},
		{
			tokenStr:   adminToken,
			permission: adminPerm,
			method:     MethodRead,
			label:      label,
		},
		{
			tokenStr:   adminToken,
			permission: adminPerm,
			method:     MethodWrite,
			label:      label,
		},
		{
			tokenStr:   adminToken,
			permission: adminPerm,
			method:     MethodDelete,
			label:      label,
		},

		// read
		{
			tokenStr:   readToken,
			permission: readPerm,
			method:     MethodRead,
			label:      label,
		},

		// write
		{
			tokenStr:   writeToken,
			permission: writePerm,
			method:     MethodWrite,
			label:      label,
		},

		// delete
		{
			tokenStr:   delToken,
			permission: delPerm,
			method:     MethodDelete,
			label:      label,
		},

		// all but admin token
		{
			tokenStr:   allButAdminToken,
			permission: allButAdminPerm,
			method:     MethodRead,
			label:      label,
		},
		{
			tokenStr:   allButAdminToken,
			permission: allButAdminPerm,
			method:     MethodWrite,
			label:      label,
		},
		{
			tokenStr:   allButAdminToken,
			permission: allButAdminPerm,
			method:     MethodDelete,
			label:      label,
		},
	}
	runValidateJWT(verifier, validCases, func(i int, c validateJWTCase, err error) {
		require.NoError(err, fmt.Sprintf(
			"case(%d) should be valid\nPermission(r,w,del,admin): %v\nMethod: %v",
			i+1, c.permission, c.method))
	})

	// test invalid cases

	invalidCases := []validateJWTCase{
		// read token shouldn't have permission on other methods
		{
			tokenStr:   readToken,
			permission: readPerm,
			method:     MethodWrite,
			label:      label,
		},
		{
			tokenStr:   readToken,
			permission: readPerm,
			method:     MethodDelete,
			label:      label,
		},
		{
			tokenStr:   readToken,
			permission: readPerm,
			method:     MethodAdmin,
			label:      label,
		},

		// write
		{
			tokenStr:   writeToken,
			permission: writePerm,
			method:     MethodRead,
			label:      label,
		},
		{
			tokenStr:   writeToken,
			permission: writePerm,
			method:     MethodDelete,
			label:      label,
		},
		{
			tokenStr:   writeToken,
			permission: writePerm,
			method:     MethodAdmin,
			label:      label,
		},

		// delete
		{
			tokenStr:   delToken,
			permission: delPerm,
			method:     MethodRead,
			label:      label,
		},
		{
			tokenStr:   delToken,
			permission: delPerm,
			method:     MethodWrite,
			label:      label,
		},
		{
			tokenStr:   delToken,
			permission: delPerm,
			method:     MethodAdmin,
			label:      label,
		},

		// all but admin
		{
			tokenStr:   allButAdminToken,
			permission: allButAdminPerm,
			method:     MethodAdmin,
			label:      label,
		},

		// empty token
		{
			tokenStr: "",
			method:   MethodAdmin,
			label:    label,
		},

		// empty context
		{
			emptyCtx: true,
			method:   MethodAdmin,
			label:    label,
		},
	}
	runValidateJWT(verifier, invalidCases, func(i int, c validateJWTCase, err error) {
		require.Error(err, fmt.Sprintf(
			"case(%d) should be invalid\nPermission(r,w,del,admin): %v\nMethod: %v",
			i+1, c.permission, c.method))
	})
}

// validateJWTCase represents a ValiddateJWT test case
type validateJWTCase struct {
	tokenStr   string
	method     Method
	label      string
	emptyCtx   bool           // if true it will skip setting token into context
	permission iyo.Permission // provides extra context to the case for feedback
	msg        string         // provides extra context for an invalid case
}

// runValidateJWT ranges over the provided validateJWT cases,
// runs ValidateJWT for each case
// and calls the validator callback
func runValidateJWT(verifier TokenVerifier, cases []validateJWTCase, validator func(caseIndex int, c validateJWTCase, err error)) {
	for i, c := range cases {
		var authCtx context.Context
		if !c.emptyCtx {
			// set token into test context
			md := metadata.New(map[string]string{
				rpctypes.MetaAuthKey: c.tokenStr,
			})
			authCtx = metadata.NewIncomingContext(context.Background(), md)
		} else {
			authCtx = context.Background()
		}

		err := verifier.ValidateJWT(authCtx, c.method, c.label)

		validator(i, c, err)
	}
}

func TestValidateNamespaceLabel(t *testing.T) {
	require := require.New(t)

	verifier, err := getTestVerifier(true)
	require.NoError(err, "failed to create test verifier")

	validCases := []string{
		"org_0stor_namespace",
		"first_0stor_gig",
	}
	runNamespaceValidator(verifier, validCases, func(c string, err error) {
		require.NoError(err, fmt.Sprintf("`%s` should be a valid case", c))
	})

	invalidCases := []string{
		"just a simple string",
		"org.0stor.namespace",
		"org_0-stor_namespace",
		"org_0stor_",
	}
	runNamespaceValidator(verifier, invalidCases, func(c string, err error) {
		require.Error(err, fmt.Sprintf("`%s` should be an invalid case", c))
	})
}

// runNamespaceValidator ranges over each test case,
// runs ValidateNamespaceLabel
// and calls the validator callback
func runNamespaceValidator(verifier TokenVerifier, cases []string, validator func(caseStr string, err error)) {
	v, ok := verifier.(*Verifier)
	if !ok {
		return
	}
	for _, c := range cases {
		err := v.validateNamespaceLabel(c)
		validator(c, err)
	}
}

func TestCheckPermissions(t *testing.T) {
	require := require.New(t)

	verifier, err := getTestVerifier(true)
	require.NoError(err, "failed to create test verifier")

	validCases := []permCase{
		{
			expectedScopes: []string{"org.0stor.ns"},
			userScopes:     []string{"user:memberof:org.0stor.ns"},
		},
		{
			expectedScopes: []string{"org.0stor.ns"},
			userScopes:     []string{"org.0stor.ns"},
		},
		{
			expectedScopes: []string{"org.0stor.ns", "org.0stor.ns.read"},
			userScopes:     []string{"org.0stor.ns"},
		},
	}
	runCheckPermissionsValidator(verifier, validCases, func(c permCase, result bool) {
		require.True(result, fmt.Sprintf(
			"following scopes should have permission\nexpected scopes: %s\n user scopes: %s",
			c.expectedScopes, c.userScopes))
	})

	invalidCases := []permCase{
		{
			expectedScopes: []string{"org.0stor.ns"},
			userScopes:     []string{"user:org.0stor.ns"},
		},
		{
			expectedScopes: []string{"org.0stor.ns"},
			userScopes:     []string{"org.0stor.ns.read", "org.0stor.ns.write", "org.0stor.ns.delete"},
		},
	}
	runCheckPermissionsValidator(verifier, invalidCases, func(c permCase, result bool) {
		require.False(result, fmt.Sprintf(
			"following scopes should NOT have permission\nexpected scopes: %s\n user scopes: %s",
			c.expectedScopes, c.userScopes))
	})
}

// permcase represents a CheckPermissions test case
type permCase struct {
	expectedScopes []string
	userScopes     []string
}

// runCheckPermissionsValidator ranges over the provided cases
// and calls the validator callback
func runCheckPermissionsValidator(verifier TokenVerifier, cases []permCase, validator func(c permCase, result bool)) {
	v, ok := verifier.(*Verifier)
	if !ok {
		return
	}
	for _, c := range cases {
		result := v.checkPermissions(c.expectedScopes, c.userScopes)
		validator(c, result)
	}
}

func TestExpectedScopes(t *testing.T) {
	require := require.New(t)
	label := "org_0stor_ns"

	verifier, err := getTestVerifier(true)
	require.NoError(err, "failed to create test verifier")

	validCases := []expScopeCase{
		{
			method: MethodAdmin,
			label:  label,
			expectedScopes: []string{
				"org.0stor.ns",
			},
		},
		{
			method: MethodRead,
			label:  label,
			expectedScopes: []string{
				"org.0stor.ns.read",
				"org.0stor.ns",
			},
		},
		{
			method: MethodWrite,
			label:  label,
			expectedScopes: []string{
				"org.0stor.ns.write",
				"org.0stor.ns",
			},
		},
		{
			method: MethodDelete,
			label:  label,
			expectedScopes: []string{
				"org.0stor.ns.delete",
				"org.0stor.ns",
			},
		},
	}

	runExpectedScopesValidator(verifier, validCases, func(i int, c expScopeCase, resultScopes []string, err error) {
		require.NoError(err, fmt.Sprintf("case(%d) should be valid", i))
		require.Equal(c.expectedScopes, resultScopes, fmt.Sprintf(
			"scopes did not match\nExpected scopes %v\nResult scopes %v",
			c.expectedScopes, resultScopes))

	})

	invalidCases := []expScopeCase{
		{
			method: MethodAdmin,
			label:  "0stor",
			msg:    "label `0stor` should be invalid",
		},
		{
			method: MethodAdmin,
			label:  "_0stor_",
			msg:    "label `_0stor_` should be invalid",
		},
		{
			method: MethodAdmin,
			label:  "org_0stor",
			msg:    "label `org_0stor` should be invalid",
		},
		{
			method: MethodAdmin,
			label:  "org_0stor_",
			msg:    "label `org_0stor_` should be invalid",
		},
	}

	runExpectedScopesValidator(verifier, invalidCases, func(i int, c expScopeCase, resultScopes []string, err error) {
		require.Error(err, fmt.Sprintf("case(%d) should be invalid\n Label: %v", i+1, c.label))
	})
}

// expScopeCase represents an ExpectedScopes test case
type expScopeCase struct {
	method         Method
	label          string
	expectedScopes []string
	msg            string // provides extra context for an invalid case
}

// runExpectedScopesValidator ranges over the provided expScope cases,
// runs ExpectedScopes for each case
// and calls the validator callback
func runExpectedScopesValidator(verifier TokenVerifier, cases []expScopeCase, validator func(caseIndex int, c expScopeCase, resultScopes []string, err error)) {
	v, ok := verifier.(*Verifier)
	if !ok {
		return
	}
	for i, c := range cases {
		result, err := v.expectedScopes(c.method, c.label)
		validator(i, c, result, err)
	}
}

func TestGetScopes(t *testing.T) {
	require := require.New(t)

	org := "org"
	namespace := "ns"

	verifier, err := getTestVerifier(true)
	require.NoError(err, "failed to create test verifier")

	adminToken := getToken(require, iyo.Permission{Admin: true},
		org, namespace)

	validCases := []getScopesCase{
		{
			tokenStr:       adminToken,
			expectedScopes: []string{"user:memberof:org.0stor.ns"},
		},
		// again to test caching
		{
			tokenStr:       adminToken,
			expectedScopes: []string{"user:memberof:org.0stor.ns"},
		},
		{
			tokenStr: getToken(require, iyo.Permission{
				Delete: true,
			}, org, namespace),
			expectedScopes: []string{"user:memberof:org.0stor.ns.delete"},
		},
		{
			tokenStr: getToken(require, iyo.Permission{
				Read:  true,
				Write: true,
			}, org, namespace),
			expectedScopes: []string{
				"user:memberof:org.0stor.ns.read",
				"user:memberof:org.0stor.ns.write",
			},
		},
		{
			tokenStr: getToken(require, iyo.Permission{
				Read:   true,
				Write:  true,
				Delete: true,
				Admin:  true,
			}, org, namespace),
			expectedScopes: []string{
				"user:memberof:org.0stor.ns.read",
				"user:memberof:org.0stor.ns.write",
				"user:memberof:org.0stor.ns.delete",
				"user:memberof:org.0stor.ns",
			},
		},
	}

	runGetScopesValidator(verifier, validCases,
		func(c getScopesCase, resultScopes []string, err error) {
			require.NoError(err)
			require.Equal(c.expectedScopes, resultScopes,
				fmt.Sprintf(
					"scopes were not as expected\n Expected scopes %v\nResult scopes: %v",
					c.expectedScopes, resultScopes))
		})

	expiredAdminToken := getTokenWithExpiration(require, iyo.Permission{Admin: true},
		-1, org, namespace)
	invalidCases := []getScopesCase{
		{
			tokenStr: expiredAdminToken,
			msg:      "token should be invalid due to expiration",
		},
		// again to test cashing
		{
			tokenStr: expiredAdminToken,
			msg:      "token should be invalid due to expiration (cached)",
		},
	}

	runGetScopesValidator(verifier, invalidCases,
		func(c getScopesCase, resultScopes []string, err error) {
			require.Error(err, c.msg)
		})

}

// getScopesCase represents a GetScopes test case
type getScopesCase struct {
	tokenStr       string
	expectedScopes []string
	msg            string // provides extra context for an invalid case
}

// runGetScopesValidator ranges over the provided getScopes cases,
// runs GetScopes for each case
// and calls the validator callback
func runGetScopesValidator(verifier TokenVerifier, cases []getScopesCase,
	validator func(c getScopesCase, resultScopes []string, err error)) {
	v, ok := verifier.(*Verifier)
	if !ok {
		return
	}
	for _, c := range cases {
		scopes, err := v.getScopes(c.tokenStr)
		validator(c, scopes, err)
	}
}

// getToken returns a JWT token for testing with the testing private key
// with default expiration of 24 hours
func getToken(require *require.Assertions, perm iyo.Permission, org, namespace string) string {
	return getTokenWithExpiration(require, perm, 24, org, namespace)
}

// getTokenWithExpiration returns a JWT token for testing with the testing private key
// with provided expiration
func getTokenWithExpiration(require *require.Assertions, perm iyo.Permission, hoursValid time.Duration, org, namespace string) string {
	b, err := ioutil.ReadFile(testPrivKeyPath)
	require.NoError(err, "failed to read private key")

	key, err := jwtgo.ParseECPrivateKeyFromPEM(b)
	require.NoError(err, "failed to parse private key")

	token, err := createJWT(hoursValid, org, namespace, perm, key)
	if err != nil {
		require.NoError(err, "failed to create iyo token")
	}

	return token
}

// CreateJWT generate a JWT that can be used for testing
func createJWT(hoursValid time.Duration, organization, namespace string, perm iyo.Permission, jwtSingingKey crypto.PrivateKey) (string, error) {
	claims := jwtgo.MapClaims{
		"exp":   time.Now().Add(time.Hour * hoursValid).Unix(),
		"scope": perm.Scopes(organization, "0stor."+namespace),
	}

	token := jwtgo.NewWithClaims(jwtgo.SigningMethodES384, claims)
	return token.SignedString(jwtSingingKey)
}
