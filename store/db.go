package main

import (
	"github.com/zaibon/badger/badger"
	"encoding/json"
	"fmt"
	"os"
)

type Settings struct {
	Dirs struct {
		Meta string `json:"meta"`
		Data string `json:"data"`
	}`json:"dirs"`
}


var settings Settings

type Badger struct {
	store *badger.KV
}

func (b *Badger) Init(){
	configFile, err := os.Open("config.json")
	defer configFile.Close()
	if err != nil{
		fmt.Println(err.Error())
		os.Exit(1)
	}

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&settings)
	os.MkdirAll(settings.Dirs.Meta, 0774)
	os.MkdirAll(settings.Dirs.Data, 0774)
}
/* Constructor */
func (b *Badger) New() *Badger{
	opts := badger.DefaultOptions
	opts.Dir = settings.Dirs.Meta
	opts.ValueDir = settings.Dirs.Data

	return &Badger{
		store:badger.NewKV(&opts),
	}
}

/* Close connection */
func (b *Badger) Close(){
	b.store.Close()
}

/* Get */
func (b *Badger) Get(key []byte) []byte{
	v, _ := b.store.Get(key)
	return v
}

/* Set */

func (b *Badger) Set(key, val []byte){
	b.store.Set(key, val)
}

