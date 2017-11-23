package jwt

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/client/itsyouonline"
)

func BenchmarkJWTCache(b *testing.B) {
	require := require.New(b)

	token := getTokenBench(require)
	verifier, err := getTestVerifier(false)
	require.NoError(err, "failed to create jwt verifier")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, err := verifier.getScopes(token)
		require.NoError(err, "getScopes failed")
	}
}

func BenchmarkJWTWithoutCache(b *testing.B) {
	require := require.New(b)

	token := getTokenBench(require)
	verifier, err := getTestVerifier(false)
	require.NoError(err, "failed to create jwt verifier")

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _, err := verifier.checkJWTGetScopes(token)
		if err != nil {
			require.NoError(err, "getScopes failed")
		}
	}
}

func getTokenBench(require *require.Assertions) string {
	return getToken(require, itsyouonline.Permission{
		Read:   true,
		Write:  true,
		Delete: true,
	}, "org", "namespace")
}
