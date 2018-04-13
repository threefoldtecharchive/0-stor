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

package daemon

import (
	"io/ioutil"

	"github.com/zero-os/0-stor/client"
	"github.com/zero-os/0-stor/client/metastor/encoding"

	yaml "gopkg.in/yaml.v2"
)

// Config defines a 0-stor daemon configuration
type Config struct {
	client.Config `yaml:",inline"`

	// MetaStor defines the configuration for the metadata server.
	MetaStor MetaStorConfig `yaml:"metastor"`
}

// MetaStorConfig is used to configure the metastor client.
type MetaStorConfig struct {
	// DB defines configuration needed to create metastor DB object.
	// This configuration is not optional
	DB MetaStorDBConfig `yaml:"db"`
	// Encryption defines the encryption processor used to
	// encrypt and decrypt the metadata prior to storage and decoding.
	//
	// This configuration is optional,
	// and when not given, no encryption is used.
	// Even though encryption is disabled by default,
	// it is recommended to use it if you can.
	Encryption client.MetaStorEncryptionConfig `yaml:"encryption" json:"encryption"`

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

// MetaStorDBConfig defines configuration needed to creates
// metastor DB object
type MetaStorDBConfig struct {
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config"`
}

// ReadConfig reads the configuration from a file.
// NOTE that it isn't validated, this will be done automatically,
// when you use the config to create a 0-stor daemon
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
