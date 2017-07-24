package config

import (
	"fmt"
	"io"

	"gopkg.in/validator.v2"
	"gopkg.in/yaml.v2"

	"github.com/zero-os/0-stor-lib/chunker"
	"github.com/zero-os/0-stor-lib/compress"
	"github.com/zero-os/0-stor-lib/distribution"
	"github.com/zero-os/0-stor-lib/encrypt"
	"github.com/zero-os/0-stor-lib/hash"
	"github.com/zero-os/0-stor-lib/replication"
)

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

func (p Pipe) CreateWriter(w io.Writer, shards []string) (io.Writer, error) {
	if p.Action != "write" {
		return nil, fmt.Errorf("not write action")
	}
	switch p.Type {
	case chunkerStr:
		//return p.createChunkerWriter(w)
		panic("chunker is not supported by pipe.CreateWriter")
	case compressStr:
		return p.createCompressWriter(w)
	case distributionStr:
		return p.createStorDistributor(shards)
	case encryptStr:
		return p.createEncryptWriter(w)
	case hashStr:
		return p.createHashWriter(w)
	default:
		return nil, fmt.Errorf("invalid type")
	}
}

func (p Pipe) createCompressWriter(w io.Writer) (compress.Writer, error) {
	conf := p.Config.(compress.Config)
	return compress.NewWriter(conf, w)
}

func (p Pipe) createEncryptWriter(w io.Writer) (*encrypt.Writer, error) {
	conf := p.Config.(encrypt.Config)
	return encrypt.NewWriter(w, conf)
}

func (p Pipe) createHashWriter(w io.Writer) (*hash.Writer, error) {
	conf := p.Config.(hash.Config)
	return hash.NewWriter(w, conf)
}

func (p Pipe) createStorDistributor(shards []string) (*distribution.StorDistributor, error) {
	conf := p.Config.(distribution.Config)
	return distribution.NewStorDistributor(shards, conf)
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
	case chunkerStr:
		var conf chunker.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case compressStr:
		var conf compress.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case encryptStr:
		var conf encrypt.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case hashStr:
		var conf hash.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case replicationStr:
		var conf replication.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case distributionStr:
		var conf distribution.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf
	default:
		panic("invalid type:" + p.Type)

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
