package config

import (
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/validator.v2"
	"gopkg.in/yaml.v2"

	"github.com/zero-os/0-stor-lib/chunker"
	"github.com/zero-os/0-stor-lib/compress"
	"github.com/zero-os/0-stor-lib/distribution"
	"github.com/zero-os/0-stor-lib/encrypt"
	"github.com/zero-os/0-stor-lib/hash"
	"github.com/zero-os/0-stor-lib/replication"
)

var (
	validPipes = map[string]struct{}{
		"chunker":      struct{}{},
		"compress":     struct{}{},
		"distribution": struct{}{},
		"encrypt":      struct{}{},
		"hash":         struct{}{},
		"replication":  struct{}{},
	}
)

// Config defines 0-stor client config
type Config struct {
	Namespace string `yaml:"namespace"`
	Pipes     []Pipe `yaml:"pipes"`
}

// NewFromReader creates Config from a reader
func NewFromReader(r io.Reader) (*Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var conf Config

	// unmarshall
	if err := yaml.Unmarshal(b, &conf); err != nil {
		return nil, err
	}

	// validate
	if err := validator.Validate(conf); err != nil {
		return nil, err
	}

	// do post processing to each pipe
	for i, pipe := range conf.Pipes {
		if err := pipe.validate(); err != nil {
			return nil, err
		}

		if err := pipe.setConfigType(); err != nil {
			return nil, err
		}
		conf.Pipes[i] = pipe
	}
	return &conf, nil
}

// Write writes the config to the given writer
func (conf *Config) Write(w io.Writer) error {
	b, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	_, err = w.Write(b)
	return err
}

// Pipe defines each 0-stor client pipe
type Pipe struct {
	Name string `yaml:"name"`

	// Type of this pipe, must be one of:
	// chunker, compress, distribution, encrypt, hash, replication
	Type string `yaml:"type" validate:"nonzero"`

	// Action to be performed: `write` or `read`.
	Action string      `yaml:"action" validate:"nonzero"`
	Config interface{} `yaml:"config"`
}

// set p.Config to proper type.
// because by default the parser is going to
// set tyhe type as map[string]interface{}
func (p *Pipe) setConfigType() error {
	b, err := yaml.Marshal(p.Config)
	if err != nil {
		return err
	}

	switch p.Type {
	case "chunker":
		var conf chunker.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case "compress":
		var conf compress.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case "distribution":
		var conf distribution.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case "encrypt":
		var conf encrypt.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case "hash":
		var conf hash.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case "replication":
		var conf replication.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	default:
		panic("invalid type")

	}
	return nil
}

// validate a pipe config
func (p Pipe) validate() error {
	if err := validator.Validate(p); err != nil {
		return err
	}

	if _, ok := validPipes[p.Type]; !ok {
		return fmt.Errorf("invalid pipe: %v", p.Type)
	}

	if p.Action != "write" && p.Action != "read" {
		return fmt.Errorf("invalid action: %v", p.Action)
	}

	return nil
}
