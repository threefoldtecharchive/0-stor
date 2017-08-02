package itsyouonline

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/itsyouonline/identityserver/clients/go/itsyouonline"
)

const (
	accessTokenURI = "https://itsyou.online/v1/oauth/access_token?response_type=id_token"
)

var (
	errNoPermission = errors.New("no permission")
)

// Client defines itsyouonline client which is designed to help 0-stor user.
// It is not replacement for official itsyouonline client
type Client struct {
	org        string
	clientID   string
	secret     string
	httpClient http.Client
	iyoClient  *itsyouonline.Itsyouonline
}

// NewClient creates new client
func NewClient(org, clientID, secret string) *Client {
	return &Client{
		org:       org,
		clientID:  clientID,
		secret:    secret,
		iyoClient: itsyouonline.NewItsyouonline(),
	}
}

// CreateJWT creates itsyouonline JWT token with these scopes:
// - org.namespace.read if perm.Read is true
// - org.namespace.write if perm.Write is true
// - org.namespace.delete if perm.Delete is true
func (c *Client) CreateJWT(namespace string, perm Permission) (string, error) {
	qp := map[string]interface{}{
		"grant_type":    "client_credentials",
		"client_id":     c.clientID,
		"client_secret": c.secret,
	}

	// build scopes query
	scopes := perm.scopes(c.org, "0stor" + "." + namespace)
	if len(scopes) == 0 {
		return "", errNoPermission
	}
	qp["scope"] = strings.Join(scopes, ",")

	// create the request
	req, err := http.NewRequest("POST", accessTokenURI, nil)
	if err != nil {
		return "", err
	}
	req.URL.RawQuery = buildQueryString(req, qp)

	// do request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// read response
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed to get access token, response code = %v", resp.StatusCode)
	}

	b, err := ioutil.ReadAll(resp.Body)
	return string(b), err

}

// CreateNamespace creates namespace as itsyouonline organization
// It creates these organizations:
// - org.namespace.read
// - org.namespace.write
// - org.namespace.write
func (c *Client) CreateNamespace(namespace string) error {
	_, _, _, err := c.iyoClient.LoginWithClientCredentials(c.clientID, c.secret)
	if err != nil {
		return fmt.Errorf("login failed:%v", err)
	}

	// create namespace org
	namespaceID := c.org + "." + "0stor"
	org := itsyouonline.Organization{
		Globalid: namespaceID,
	}
	_, resp, err := c.iyoClient.Organizations.CreateNewSubOrganization(c.org, org, nil, nil)
	if err != nil {
		return fmt.Errorf("code=%v, err=%v", resp.StatusCode, err)
	}


	// cretate 0stor suborganization

	org = itsyouonline.Organization{
		Globalid: namespaceID + "." + namespace,
	}
	_, resp, err = c.iyoClient.Organizations.CreateNewSubOrganization(namespaceID, org, nil, nil)

	if err != nil {
		return fmt.Errorf("code=%v, err=%v", resp.StatusCode, err)
	}

	namespaceID = namespaceID + "." + namespace

	// create permission org
	perm := Permission{
		Read:   true,
		Delete: true,
		Write:  true,
	}
	for _, perm := range perm.perms() {
		org := itsyouonline.Organization{
			Globalid: namespaceID + "." + perm,
		}
		_, resp, err := c.iyoClient.Organizations.CreateNewSubOrganization(namespaceID , org, nil, nil)
		if err != nil {
			return fmt.Errorf("code=%v, err=%v", resp.StatusCode, err)
		}
	}
	return nil
}

func buildQueryString(req *http.Request, qs map[string]interface{}) string {
	q := req.URL.Query()

	for k, v := range qs {
		q.Add(k, fmt.Sprintf("%v", v))
	}
	return q.Encode()
}
