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
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/threefoldtech/0-stor/client/datastor/pipeline"

	yaml "gopkg.in/yaml.v2"
)

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
	// Password defines the optional 0-db password.
	Password string `yaml:"password" json:"password"`
	// Namespace defines the label (ID of namespace),
	// to be used for all read/write/delete operations.
	Namespace string `yaml:"namespace" json:"namespace"`

	// DataStor defines the configuration for the zstordb data shards (servers),
	// at least one zstordb shard is given, but more might be required,
	// if you define a distributed storage configuration in the pipeline config.
	DataStor DataStorConfig `yaml:"datastor" json:"datastor"`

	// MetaStor defines the configuration for the metadata shards (servers).
	// For now only an ETCD cluster is supported using this config.
	//MetaStor MetaStorConfig `yaml:"metastor" json:"metastor"`
}

// DataStorConfig is used to configure a zstordb cluster.
type DataStorConfig struct {
	// Shards defines the Listed shards, at least one listed shard is required
	Shards []string `yaml:"shards" json:"shards"` // required

	// Pipeline defines the object read/write pipeline configuration
	// for this 0-stor client. It defines how to structure,
	// process, identify and store all data to be written,
	// and that same configuration is required to read the data back.
	Pipeline pipeline.Config `yaml:"pipeline" json:"pipeline"`

	// TLS defines the optional global TLS config,
	// which is used for all lised and unlisted datastor shards, in case it is given.
	TLS DataStorTLSConfig `yaml:"tls" json:"tls"`
}

// DataStorTLSConfig is used to config the global TLS config used
// for all listed and unlisted datastor shards.
type DataStorTLSConfig struct {
	// has to be true in order to enable this config
	Enabled bool `yaml:"enabled" json:"enabled"`
	// when not given the TLS implemenation will skip certification verification,
	// exposing the client to man-in-the-middle attacks
	ServerName string `yaml:"server" json:"server"`
	// when not given, the system CA will be used
	RootCA string `yaml:"root_ca" json:"root_ca"`

	// optional min/max TLS versions, limiting the
	// accepted TLS version used by the server
	MinVersion TLSVersion `yaml:"min_version" json:"min_version"`
	MaxVersion TLSVersion `yaml:"max_version" json:"max_version"`
}

// TLSVersion defines a TLS Version,
// usable to restrict the possible TLS Versions.
type TLSVersion uint8

const (
	// UndefinedTLSVersion defines an undefined TLS Version,
	// which can can be used to signal the desired use of a default TLS Version
	UndefinedTLSVersion TLSVersion = iota
	// TLSVersion12 defines TLS version 1.2,
	// and is also the current default TLS Version.
	TLSVersion12
	// TLSVersion11 defines TLS version 1.1
	TLSVersion11
	// TLSVersion10 defines TLS version 1.0,
	// but should not be used, unless you have no other option.
	TLSVersion10

	_MaxTLSVersion = TLSVersion10
)

var (
	// important that this order stays in sync with
	// the order of constants definitions from above!
	_TLSVersionStrings = []string{
		"TLS12",
		"TLS11",
		"TLS10",
	}
	_TlsVersionValues = []uint16{
		tls.VersionTLS12,
		tls.VersionTLS11,
		tls.VersionTLS10,
	}
)

// String implements Stringer.String
func (v TLSVersion) String() string {
	if v == UndefinedTLSVersion || v > _MaxTLSVersion {
		return ""
	}
	return _TLSVersionStrings[v-1]
}

// MarshalText implements encoding.TextMarshaler.MarshalText
func (v TLSVersion) MarshalText() (text []byte, err error) {
	if v > _MaxTLSVersion {
		return nil, fmt.Errorf("invalid in-memory TLS version %d", uint8(v))
	}
	if v == UndefinedTLSVersion {
		return nil, nil
	}
	return []byte(_TLSVersionStrings[v-1]), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText
func (v *TLSVersion) UnmarshalText(text []byte) error {
	str := strings.ToUpper(string(text))
	if len(str) == 0 {
		*v = UndefinedTLSVersion
		return nil
	}
	for index, verStr := range _TLSVersionStrings {
		if verStr == str {
			*v = TLSVersion(index + 1)
			return nil
		}
	}
	return fmt.Errorf("TLS version %s not recognized or supported", str)
}

// VersionTLSOrDefault returns the tls.VersionTLS value,
// which corresponds to this TLSVersion.
// Or it returns the given default TLS Version in case
// this TLS Version is undefined.
func (v TLSVersion) VersionTLSOrDefault(def uint16) uint16 {
	if v > _MaxTLSVersion {
		panic(fmt.Sprintf("invalid TLS Version: %d", v))
	}
	if v == UndefinedTLSVersion {
		return def
	}
	return _TlsVersionValues[v-1]
}
