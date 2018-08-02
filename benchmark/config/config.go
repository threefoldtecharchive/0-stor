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

//Package config provides the config types and helper functions for zstorbench
package config

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/threefoldtech/0-stor/daemon"

	validator "gopkg.in/validator.v2"
	yaml "gopkg.in/yaml.v2"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// ClientConf represents a client banchmark config
type ClientConf struct {
	Scenarios map[string]Scenario `yaml:"scenarios" validate:"nonzero"`
}

// validate validates a ClientConf
func (clientConf *ClientConf) validate() error {
	for _, sc := range clientConf.Scenarios {
		err := sc.validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// Scenario represents a scenario
type Scenario struct {
	ZstorConf daemon.Config   `yaml:"zstor" validate:"nonzero"`
	BenchConf BenchmarkConfig `yaml:"benchmark" validate:"nonzero"`
}

// validate validates a scenario
func (sc *Scenario) validate() error {
	return sc.BenchConf.validate()
}

// SetupClientConfig sets up the client.Client for a benchmark.
// Sets random namespace if empty
func SetupClientConfig(c *daemon.Config) {
	// set namespace if not provided
	if c.Namespace == "" {
		c.Namespace = "b-" + randomSuffix(4)
	}
}

func randomSuffix(n int) string {
	chars := make([]uint8, n)
	for i := range chars {
		chars[i] = uint8(48 + rand.Intn(10))
	}
	return string(chars)
}

// BenchmarkConfig represents benchmark configuration
type BenchmarkConfig struct {
	Method     BencherMethod `yaml:"method"`
	Output     string        `yaml:"result_output"`
	Duration   int           `yaml:"duration"`
	Operations int           `yaml:"operations"`
	Clients    int           `yaml:"clients"`
	KeySize    int           `yaml:"key_size" validate:"nonzero"`
	ValueSize  int           `yaml:"value_size" validate:"nonzero"`
}

// validate validates a BenchmarkConfig
func (bc *BenchmarkConfig) validate() error {
	if bc.Duration <= 0 && bc.Operations <= 0 {
		return fmt.Errorf("no duration or operations was provided")
	}

	return nil
}

// UnmarshalYAML returns client config from a given reader
func UnmarshalYAML(b []byte) (*ClientConf, error) {
	clientConf := &ClientConf{}

	// unmarshal
	if err := yaml.Unmarshal(b, clientConf); err != nil {
		return nil, err
	}

	// validate
	err := validator.Validate(clientConf)
	if err != nil {
		return nil, err
	}

	if err := clientConf.validate(); err != nil {
		return nil, err
	}

	return clientConf, nil
}

// BencherMethod represents the method of benchmarking
type BencherMethod uint8

const (
	// BencherRead represents a reading benchmark
	BencherRead BencherMethod = iota
	// BencherWrite represents a writing benchmark
	BencherWrite
)

const _BencherMethodStrings = "readwrite"

var (
	_BencherMethodEnumToStringMapping = map[BencherMethod]string{
		BencherRead:  _BencherMethodStrings[:4],
		BencherWrite: _BencherMethodStrings[4:],
	}
	_BencherMethodStringToEnumMapping = map[string]BencherMethod{
		_BencherMethodStrings[:4]: BencherRead,
		_BencherMethodStrings[4:]: BencherWrite,
	}
)

// String implements Stringer.String
func (bm BencherMethod) String() string {
	return _BencherMethodEnumToStringMapping[bm]
}

// MarshalText implements encoding.TextMarshaler.MarshalText
func (bm BencherMethod) MarshalText() ([]byte, error) {
	str := bm.String()
	if str == "" {
		return nil, fmt.Errorf("'%v' is not a valid BencherMethod value", bm)
	}
	return []byte(str), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.UnmarshalText
func (bm *BencherMethod) UnmarshalText(text []byte) error {
	var ok bool
	*bm, ok = _BencherMethodStringToEnumMapping[strings.ToLower(string(text))]
	if !ok {
		return fmt.Errorf("'%s' is not a valid BencherMethod string", text)
	}
	return nil
}
