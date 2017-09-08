This folder contains a goreman Procfile that start 4 0-stor servers and a cluster of 3 etcd.

The config.yaml file in this directory contains the proper configuration for the 0-stor and etcd cluster

## Goreman

Clone of foreman written in golang.

https://github.com/ddollar/foreman

## install goreman

    go get github.com/mattn/goreman

## Start the 0-stor servers and etcd

    goreman start