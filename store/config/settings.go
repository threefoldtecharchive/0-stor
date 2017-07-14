package config

import (
	"encoding/json"
	"os"

	log "github.com/Sirupsen/logrus"
)

type Settings struct {
	DebugLog   bool `json:"debug"`
	DisableJWT bool `json:"disable_jwt"`

	BindAddress string `json:"bind"`

	DB struct {
		Dirs struct {
			Meta string `json:"meta"`
			Data string `json:"data"`
		} `json:"dirs"`
	} `json:db`
}

func (s *Settings) Load(path string) error {
	conf, err := os.Open(path)

	defer conf.Close()

	if err != nil {
		log.Errorln(err.Error())
		return err
	}

	if err := json.NewDecoder(conf).Decode(s); err != nil {
		log.Errorln(err.Error())
		return err
	}

	return nil
}
