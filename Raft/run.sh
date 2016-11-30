#!/bin/sh
killall xterm
go build -o Server1 Raftn1.go
go build -o Server2 Raftn1.go
go build -o Server3 Raftn1.go
go build -o Server4 Raftn1.go
go build -o Server5 Raftn1.go
xterm -e ./Server1 1 server1&
xterm -e ./Server2 2 server2&
xterm -e ./Server3 3 server3&
xterm -e ./Server4 4 server4&
xterm -e ./Server5 5 server5&