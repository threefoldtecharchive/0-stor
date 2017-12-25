#!/bin/bash

go get -u github.com/gordonklaus/ineffassign

ineffassign client server cmd
