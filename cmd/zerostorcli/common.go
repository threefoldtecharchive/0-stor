package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/itsyouonline"
)

func getClient(c *cli.Context) (*client.Client, error) {
	policy, err := readPolicy()
	if err != nil {
		return nil, err
	}
	// create client
	cl, err := client.New(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	return cl, nil
}

func getNamespaceManager(c *cli.Context) (itsyouonline.IYOClient, error) {
	policy, err := readPolicy()
	if err != nil {
		return nil, err
	}

	return itsyouonline.NewClient(policy.Organization, policy.IYOAppID, policy.IYOSecret), nil
}

func readPolicy() (client.Policy, error) {
	// read config
	f, err := os.Open(confFile)
	if err != nil {
		return client.Policy{}, err
	}

	return client.NewPolicyFromReader(f)
}
