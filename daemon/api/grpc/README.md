# gRPC Daemon

gRPC Daemon provides gRPC interface to the 0-stor client.
The gRPC interface specification can be found at [daemon.proto](./schema/daemon.proto).

It can be started by using `ztor daemon` command.

All 0-stor client library other than in Go languange should use this daemon to avoid re-implementing the 
quite complex logic of the 0-stor client.
