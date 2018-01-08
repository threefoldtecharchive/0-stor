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
	"io/ioutil"

	"github.com/zero-os/0-stor/client/datastor/pipeline"
	"github.com/zero-os/0-stor/client/itsyouonline"
	"github.com/zero-os/0-stor/client/metastor/encoding"
	"github.com/zero-os/0-stor/client/processing"

	yaml "gopkg.in/yaml.v2"
)

// TODO: handle configuration using https://github.com/spf13/viper

// ReadConfig reads the configuration from a file.
// NOTE that it isn't validated, this will be done automatically,
// when you use the config to create a 0-stor client.
func ReadConfig(path string) (*Config, error) {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// for now we only support YAML
	var cfg Config
	if err = yaml.Unmarshal(bytes, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Config defines the configuration of the 0-stor client.
// It configures everything from namespaces, permissions,
// storage clusters, as well as the entire read/write pipeline,
// used to read and write data.
type Config struct {
	// IYO defines the IYO (itsyou.online) configuration,
	// used for this 0-stor client.
	IYO itsyouonline.Config `yaml:"iyo" json:"iyo"`
	// Namespace defines the label (ID of namespace),
	// to be used for all read/write/delete operations.
	Namespace string `yaml:"namespace" json:"namespace"`

	// DataStor defines the configuration for the zstordb data shards (servers),
	// at least one zstordb shard is given, but more might be required,
	// if you define a distributed storage configuration in the pipeline config.
	DataStor DataStorConfig `yaml:"datastor" json:"datastor"`

	// MetaStor defines the configuration for the metadata shards (servers).
	// For now only an ETCD cluster is supported using this config.
	MetaStor MetaStorConfig `yaml:"metastor" json:"metastor"`
}

// DataStorConfig is used to configure a zstordb cluster.
type DataStorConfig struct {
	Shards []string `yaml:"shards" json:"shards"` // required

	// Pipeline defines the object read/write pipeline configuration
	// for this 0-stor client. It defines how to structure,
	// process, identify and store all data to be written,
	// and that same configuration is required to read the data back.
	Pipeline pipeline.Config `yaml:"pipeline" json:"pipeline"`
}

// MetaStorConfig is used to configure the metastor client.
// TODO: remove this fixed/non-dynamic struct
type MetaStorConfig struct {
	// Database defines the configuration of the database backend,
	// used to set, get and delete metadata into valid KV database.
	///
	// This is the only configuration which is required,
	// currently only an ETCD-backed database is supported,
	// when using this Config. Should you desire another backend,
	// such as badger, you'll need to create the client manually,
	// for now.
	Database MetaStorETCDConfig `yaml:"db" json:"db"`

	// Encryption defines the encryption processor used to
	// encrypt and decrypt the metadata prior to storage and decoding.
	//
	// This configuration is optional,
	// and when not given, no encryption is used.
	// Even though encryption is disabled by default,
	// it is recommended to use it if you can.
	Encryption MetaStorEncryptionConfig `yaml:"encryption" json:"encryption"`

	// Encoding defines the encoding type,
	// used to marshal the metadata to binary from, and vice versa.
	//
	// This property is optional, and by default protobuf is used.
	// Protobuf is also the only standard option available,
	// however using encoding.RegisterMarshalFuncPair,
	// you'll be able to register (or overwrite an existing) MarshalFuncPair,
	// and thus support any encoder you wish to use.
	Encoding encoding.MarshalType `yaml:"encoding" json:"encoding"` // optional (proto by default)
}

// MetaStorEncryptionConfig defines the configuration used to create an
// encrypter-decrypter Processor, used to encrypt the metadata prior to storage,
// and decrypting it right after fetching it back from the database.
type MetaStorEncryptionConfig struct {
	// Private key, the specific required length
	// is defined by the type of Encryption used.
	//
	// This key will also used by the crypto-hashing algorithm given,
	// if you did not define a separate key within the hashing configuration.
	PrivateKey string `yaml:"private_key" json:"private_key"`

	// The type of encryption algorithm to use,
	// defining both the encrypting and decrypting logic.
	// The string value (representing the encryption algorithm type), is case-insensitive.
	//
	// By default no type is used, disabling encryption,
	// encryption gets enabled as soon as a private key gets defined.
	// All standard types available are: AES
	//
	// Valid Key sizes for AES are: 16, 24 and 32 bytes
	// The recommended private key size is 32 bytes, this will select/use AES_256.
	//
	// In case you've registered a custom encryption algorithm,
	// or have overridden a standard encryption algorithm, using `processing.RegisterEncrypterDecrypter`
	// you'll be able to use that encrypter-decrypting, by providing its (stringified) type here.
	Type processing.EncryptionType `yaml:"type" json:"type"`
}

// MetaStorETCDConfig is used to configure/create an ETCD-cluster client,
// and use it as the database backend for the metastor client.
type MetaStorETCDConfig struct {
	Endpoints []string `yaml:"endpoints" json:"endpoints"` // required
}
