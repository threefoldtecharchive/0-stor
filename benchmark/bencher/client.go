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

package bencher

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/threefoldtech/0-stor/client"
	"github.com/threefoldtech/0-stor/client/datastor"
	"github.com/threefoldtech/0-stor/client/datastor/pipeline"
	"github.com/threefoldtech/0-stor/client/datastor/zerodb"
	"github.com/threefoldtech/0-stor/client/metastor"
	metaDB "github.com/threefoldtech/0-stor/client/metastor/db"
	"github.com/threefoldtech/0-stor/client/metastor/db/test"
	db_utils "github.com/threefoldtech/0-stor/client/metastor/db/utils"
	"github.com/threefoldtech/0-stor/client/metastor/encoding"
	"github.com/threefoldtech/0-stor/client/processing"
	"github.com/threefoldtech/0-stor/daemon"

	log "github.com/sirupsen/logrus"
)

// newClientFromConfig creates a new zstor client from provided config
// if Metastor shards are empty, it will use an in memory metadata server
func newClientFromConfig(cfg *daemon.Config, jobCount int) (*client.Client, *metastor.Client, error) {
	// create datastor cluster
	datastorCluster, err := createDataClusterFromConfig(cfg)
	if err != nil {
		return nil, nil, err
	}

	// create data pipeline, using our datastor cluster
	dataPipeline, err := pipeline.NewPipeline(cfg.DataStor.Pipeline, datastorCluster, jobCount)
	if err != nil {
		return nil, nil, err
	}

	if cfg.MetaStor == nil {
		return nil, nil, fmt.Errorf("benchmarker requires metastore")
	}

	// if no metadata shards are given, return an error,
	// as we require a metastor client
	// create metastor client
	metastorClient, err := createMetastorClientFromConfig(cfg.Namespace, cfg.MetaStor)
	if err != nil {
		return nil, nil, err
	}

	return client.NewClient(metastorClient, dataPipeline), metastorClient, nil
}

func createDataClusterFromConfig(cfg *daemon.Config) (datastor.Cluster, error) {
	// optionally create the global datastor TLS config
	tlsConfig, err := createTLSConfigFromDatastorTLSConfig(&cfg.DataStor.TLS)
	if err != nil {
		return nil, err
	}

	return zerodb.NewCluster(cfg.DataStor.Shards, cfg.Password, cfg.Namespace, tlsConfig, cfg.DataStor.Spreading)
}

func createMetastorClientFromConfig(namespace string, cfg *daemon.MetaStorConfig) (*metastor.Client, error) {
	if len(cfg.DB.Type) == 0 {
		// if no config, return a test metadata server (in-memory)
		log.Debug("Using in-memory metadata server")
		return createMetastorClientFromConfigAndDatabase(namespace, cfg, test.New())
	}

	db, err := db_utils.NewMetaStorDB(cfg.DB.Type, cfg.DB.Config)
	if err != nil {
		return nil, err
	}

	// create the metastor client and the rest of its components
	return createMetastorClientFromConfigAndDatabase(namespace, cfg, db)
}

func createMetastorClientFromConfigAndDatabase(namespace string, cfg *daemon.MetaStorConfig, db metaDB.DB) (*metastor.Client, error) {
	var (
		err    error
		config = metastor.Config{Database: db}
	)

	// create the metadata encoding func pair
	config.MarshalFuncPair, err = encoding.NewMarshalFuncPair(cfg.Encoding)
	if err != nil {
		return nil, err
	}

	if len(cfg.Encryption.PrivateKey) == 0 {
		// create potentially insecure metastor storage
		return metastor.NewClientFromConfig([]byte(namespace), config)
	}

	// create the constructor which will create our encrypter-decrypter when needed
	config.ProcessorConstructor = func() (processing.Processor, error) {
		return processing.NewEncrypterDecrypter(
			cfg.Encryption.Type, []byte(cfg.Encryption.PrivateKey))
	}
	// ensure the constructor is valid,
	// as most errors (if not all) are static, and will only fail due to the given input,
	// meaning that if it can be created it now, it should be fine later on as well
	_, err = config.ProcessorConstructor()
	if err != nil {
		return nil, err
	}

	// create our full-configured metastor client,
	// including encryption support for our metadata in binary form
	return metastor.NewClientFromConfig([]byte(namespace), config)
}

func createTLSConfigFromDatastorTLSConfig(config *client.DataStorTLSConfig) (*tls.Config, error) {
	if config == nil || !config.Enabled {
		return nil, nil
	}
	tlsConfig := &tls.Config{
		MinVersion: config.MinVersion.VersionTLSOrDefault(tls.VersionTLS11),
		MaxVersion: config.MaxVersion.VersionTLSOrDefault(tls.VersionTLS12),
	}

	if config.ServerName != "" {
		tlsConfig.ServerName = config.ServerName
	} else {
		log.Warning("TLS is configured to skip verification of certs, " +
			"making the client susceptible to man-in-the-middle attacks!!!")
		tlsConfig.InsecureSkipVerify = true
	}

	if config.RootCA == "" {
		var err error
		tlsConfig.RootCAs, err = x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to create datastor TLS config: %v", err)
		}
	} else {
		tlsConfig.RootCAs = x509.NewCertPool()
		caFile, err := ioutil.ReadFile(config.RootCA)
		if err != nil {
			return nil, err
		}
		if !tlsConfig.RootCAs.AppendCertsFromPEM(caFile) {
			return nil, fmt.Errorf("error reading CA file '%s', while creating datastor TLS config: %v",
				config.RootCA, err)
		}
	}

	return tlsConfig, nil
}
