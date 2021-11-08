#!/bin/bash

go build server/relay-server.go
go build chat.go


./relay-server -sp 3001