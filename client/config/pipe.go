package config

import (
	"fmt"

	"gopkg.in/validator.v2"
	"gopkg.in/yaml.v2"

	"github.com/zero-os/0-stor/client/lib/block"
	"github.com/zero-os/0-stor/client/lib/chunker"
	"github.com/zero-os/0-stor/client/lib/compress"
	"github.com/zero-os/0-stor/client/lib/distribution"
	"github.com/zero-os/0-stor/client/lib/encrypt"
	"github.com/zero-os/0-stor/client/lib/hash"
	"github.com/zero-os/0-stor/client/lib/replication"
	"github.com/zero-os/0-stor/client/meta"
	"github.com/zero-os/0-stor/client/stor"
)

// Pipe defines each 0-stor client pipe
type Pipe struct {
	Name string `yaml:"name"`

	// Type of this pipe, must be one of:
	// chunker, compress, distribution, encrypt, hash, replication
	Type string `yaml:"type" validate:"nonzero"`

	Config interface{} `yaml:"config"`
}

// CreateBlockReader creates a block reader
func (p Pipe) CreateBlockReader(org, namespace string) (block.Reader, error) {
	switch p.Type {
	case compressStr:
		conf := p.Config.(compress.Config)
		return compress.NewReader(conf)
	case encryptStr:
		conf := p.Config.(encrypt.Config)
		return encrypt.NewReader(conf)
	case distributionStr:
		conf := p.Config.(distribution.Config)
		return distribution.NewStorRestorer(conf, org, namespace)
	case metaStr:
		conf := p.Config.(meta.Config)
		return meta.NewReader(conf)
	case storClientStr:
		conf := p.Config.(stor.Config)
		return stor.NewReader(conf, org, namespace)
	default:
		return nil, fmt.Errorf("invalid reader type:%v", p.Type)
	}
}

// CreateBlockWriter creates block writer
func (p Pipe) CreateBlockWriter(w block.Writer, org, namespace string) (block.Writer, error) {
	switch p.Type {
	case compressStr:
		conf := p.Config.(compress.Config)
		return compress.NewWriter(conf, w)
	case distributionStr:
		conf := p.Config.(distribution.Config)
		return distribution.NewStorDistributor(w, conf, org, namespace)
	case encryptStr:
		conf := p.Config.(encrypt.Config)
		return encrypt.NewWriter(w, conf)
	case hashStr:
		conf := p.Config.(hash.Config)
		return hash.NewWriter(w, conf)
	case metaStr:
		conf := p.Config.(meta.Config)
		return meta.NewWriter(w, conf)
	case storClientStr:
		conf := p.Config.(stor.Config)
		return stor.NewWriter(w, conf, org, namespace)
	default:
		return nil, fmt.Errorf("invalid writer type:%v", p.Type)
	}
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

	case metaStr:
		var conf meta.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	case storClientStr:
		var conf stor.Config
		if err := yaml.Unmarshal(b, &conf); err != nil {
			return err
		}
		p.Config = conf

	default:
		panic("invalid pipe type:" + p.Type)

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

	return nil
}
