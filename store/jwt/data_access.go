package jwt

import "github.com/zero-os/0-stor/store/rest/models"

func GenerateDataAccessTokenForUser(user string, namespaceID string, acl models.ACLEntry) (string, error) {
	return "", nil
}

// 	b := make([]byte, 60+len(namespaceID)+len(user))
//
// 	r, err := utils.GenerateRandomBytes(51)
//
// 	if err != nil {
// 		return "", err
// 	}
//
// 	copy(b[0:51], r)
// 	epoch := time.Time(s.ExpireAt).Unix()
// 	binary.LittleEndian.PutUint64(b[51:59], uint64(epoch))
// 	aclEncoded, err := acl.Encode()
// 	if err != nil {
// 		return "", err
// 	}
// 	copy(b[59:63], aclEncoded)
// 	copy(b[63:], []byte(user))
// 	token, err := base64.StdEncoding.EncodeToString(b), err
//
// 	if err != nil {
// 		return "", err
// 	}
//
// 	return token, nil
// }

func ValidateDataAccessToken(acl models.ACLEntry, token string) error {
	return nil
}

// 	bytes, err := base64.StdEncoding.DecodeString(token)
//
// 	if err != nil {
// 		return err
// 	}
// 	if len(bytes) <= 63 {
// 		return errors.New("Data access token is invalid")
// 	}
// 	now := time.Now()
// 	expiration := time.Unix(int64(binary.LittleEndian.Uint64(bytes[51:59])), 0)
//
// 	if now.After(expiration) {
// 		return errors.New("Data access token expired")
// 	}
//
// 	tokenACL := ACLEntry{}
// 	tokenACL.Decode(bytes[59:63])
//
// 	// IS Admin
// 	if tokenACL.Admin {
// 		return nil
// 	}
//
// 	// HTTP action ACL requires missing permission granted for that user
// 	if (acl.Admin && !tokenACL.Admin) ||
// 		(acl.Read && !tokenACL.Read) ||
// 		(acl.Write && !tokenACL.Write) ||
// 		(acl.Delete && !tokenACL.Delete) {
// 		return errors.New("Permission denied")
// 	}
//
// 	//tokenUser := string(bytes[63:])
//
// 	//if user != tokenUser{
// 	//	return errors.New("Invalid token for user")
// 	//}
//
// 	return nil
//
// }
