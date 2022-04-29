#!/bin/bash

#CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build server/relay-server.go
go build server/relay-server.go
go build chat.go


./relay-server -sp 3001
