package main

import "github.com/pkg/errors"

var ErrorNotFound error = errors.New("not_found")
var ErrorDB error = errors.New("database_error")
var ErrorSettings = errors.New("settings_error")
