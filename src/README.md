Play with it
============

Go into the "src", do:
  go install tfe/confs
  go run main/tfe_server.go -rules=test

To build for linux, do:
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build main/tfe_server.go

Note to be able to build for linux, Go distribution needs to be built with linux support.
