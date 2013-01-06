Introduction
============
This repo contains bunch of projects I'm playing with golang.

gostrich
============

Gostrich is a metric collection library, similar to Ostrich of Scala. It reports metrics in the
same format, either text for json. It provides counters, labels, gauges and stats.

It also provides a simple extensible Http server to expose those stats outside, a /shutdown command
to shutdown server.

rpcx
==============

A utility library for reliable RPC. It abstracts RPC in similar fashion as Finagle of Scala. At its
core is the Service interface, which provides a single function of:

  Serve(req interface{}, rep interface{}, timeout time.Duration).

Given such interface, the library provides a way to call services on multiple hosts in a round robin
fashion. It keeps track of latency stats and forwards requests to higher performing nodes. It
handles faulty services. It can shutdown a service while keep prober running or it can optionally
recreate underlying service in case of severe error.

gassandra
==============

An cassandra client library that works with github.com/samuel/go-thrift and rpcx to provide
reliable access to cassandra.

tfe
==============

TFE is a HTTP load balancer built on top of Rpcx.

main
==============

Some random services

Build
-----

Go into the "src", do:
>  go install tfe/confs
>
>  go run main/tfe_server.go -rules=test
>

To build for linux, do:
>  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build main/tfe_server.go

Note to be able to build for linux, Go distribution needs to be built with linux support.
