package embedserver

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/coreos/etcd/embed"
)

// New creates new embeddded metadata server
// which listen on unix socket
func New() ([]string, func(), error) {
	tmpDir, err := ioutil.TempDir("", "etcd")
	if err != nil {
		return nil, nil, err
	}

	cfg := embed.NewConfig()
	cfg.Dir = tmpDir

	// listen client URL
	// we use tmpDir as unix address because it is a simple
	// yet valid way to generate random string
	lcurl, err := url.Parse("unix://" + filepath.Base(tmpDir))
	if err != nil {
		return nil, nil, err
	}
	cfg.LCUrls = []url.URL{*lcurl}

	// listen peer url
	// same strategy with listen client URL
	lpDir, err := ioutil.TempDir("", "etcd")
	if err != nil {
		return nil, nil, err
	}
	lpurl, err := url.Parse("unix://" + filepath.Base(lpDir))
	if err != nil {
		return nil, nil, err
	}
	cfg.LPUrls = []url.URL{*lpurl}

	e, err := embed.StartEtcd(cfg)
	if err != nil {
		return nil, nil, err
	}

	<-e.Server.ReadyNotify()

	cleanFunc := func() {
		os.RemoveAll(tmpDir)
		os.RemoveAll(lpDir)
		e.Server.Stop()
		<-e.Server.StopNotify()
		e.Close()
	}

	conf := e.Config()

	// we currently always use this address
	return []string{conf.LCUrls[0].String()}, cleanFunc, nil
}
