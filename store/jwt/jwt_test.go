package jwt

import (
	"crypto/rand"

	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zero-os/0-stor/store/goraml"
	"github.com/zero-os/0-stor/store/rest/models"
	"github.com/zero-os/0-stor/store/utils"
)

// Override time value for tests.  Restore default value after.
func at(t time.Time, f func()) {
	jwt.TimeFunc = func() time.Time {
		return t
	}
	f()
	jwt.TimeFunc = time.Now
}

func genKey(t *testing.T) []byte {
	key := make([]byte, 2048)
	_, err := rand.Read(key)
	if err != nil {
		t.Error("error generating jwt test key: %v", err)
		t.FailNow()
	}
	return key
}

func TestReservationToken(t *testing.T) {
	key := genKey(t)
	now := time.Now()
	id, err := utils.GenerateUUID(64)
	require.NoError(t, err)

	res := models.Reservation{
		Namespace:    "ns1",
		AdminId:      "user1",
		Created:      goraml.DateTime(now),
		ExpireAt:     goraml.DateTime(now.Add(time.Hour)),
		Id:           id,
		SizeReserved: 1024,
		SizeUsed:     0,
		Updated:      goraml.DateTime(now),
	}
	tokenString, err := GenerateReservationToken(res, key)
	require.NoError(t, err)

	tt := []struct {
		name      string
		namespace string
		at        time.Time
		err       error
	}{
		{
			name:      "valid",
			namespace: "ns1",
			at:        now,
			err:       nil,
		},
		{
			name:      "expired",
			namespace: "ns1",
			at:        now.Add(time.Hour * 24),
			err:       jwt.NewValidationError("jwt expired", jwt.ValidationErrorExpired),
		},
		{
			name:      "wrong namespace",
			namespace: "ns2",
			at:        now,
			err:       ErrWrongNamespace,
		},
	}

	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			at(test.at, func() {
				_, err := ValidateReservationToken(tokenString, test.namespace, key)
				if test.err != nil {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		})
	}

}
