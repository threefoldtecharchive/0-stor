package middleware

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestA(t *testing.T) {
	token := "eyJhbGciOiJFUzM4NCIsInR5cCI6IkpXVCJ9.eyJhenAiOiJLa0Y0c3IyUll4cXVYWTZlWjVtMWtic0dTbVJRIiwiZXhwIjoxNTAwNDY1NDkzLCJpc3MiOiJpdHN5b3VvbmxpbmUiLCJzY29wZSI6WyJ1c2VyOm5hbWUiXSwidXNlcm5hbWUiOiJ6YWlib24ifQ.PiZymTVxRo1NQZVFsvsrDsdeUh6SWLKZbF6gISHfeOl5UTU2VOnGkIO1ucXXatdSNZNlq5QQFSJHvXLUmG9m8HASwPIMnwY0GlaFM3F8YDvC2AQ5CTGx-KrxegKeYW_b"

	mid := NewOauth2itsyouonlineMiddleware([]string{"user:name"})
	scopes, err := mid.checkJWTGetScope(token)
	require.NoError(t, err)
	t.Logf("%+v", scopes)
}
