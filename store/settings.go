package main

import (
	"os"
	"encoding/json"
	log "github.com/Sirupsen/logrus"
)

const(
	STORE_STATS_COLLECTION_NAME = "0@stats"
	NAMESPACE_STATS_COLlECTION_NAME_PREFIX = "0@stats_"
	NAMESPACE_COLLECTION_NAME_PREFIX = "2@_"
	NAMESPACE_RESERVATION_COLLECTION_PREFIX = "1@res_"
)

type Settings struct{

	DebugLog    bool   `json:"debug"`

	BindAddress string `json:"bind"`

	DB struct{

		Dirs struct {
			Meta string `json:"meta"`
			Data string `json:"data"`
		} `json:"dirs"`

		Iterator struct {
			PreFetchSize int `json:"pre_fetch_size"`
		} `json:"iterator"`

		Pagination struct {
			PageSize int `json:"page_size"`
		}

	}`json:badger`

	Store struct {
		Stats struct {
			Collection string `json:"collection"`
		}`json:"stats"`

	}`json:"store"`

	Namespace struct {
		Prefix string `json:"prefix"`

		Stats struct {
			Prefix string `json:"prefix"`
		}`json:"stats"`

		Reservations struct{
			Prefix string `json:"prefix"`
		}`json:"reservations"`

	}`json:namespace`
}

func (s *Settings) Load(path string) error{
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

func (s *Settings) LoadDefaults() error{
	s.Store.Stats.Collection = STORE_STATS_COLLECTION_NAME
	s.Namespace.Stats.Prefix = NAMESPACE_STATS_COLlECTION_NAME_PREFIX
	s.Namespace.Reservations.Prefix = NAMESPACE_RESERVATION_COLLECTION_PREFIX
	s.Namespace.Prefix = NAMESPACE_COLLECTION_NAME_PREFIX
	return nil
}