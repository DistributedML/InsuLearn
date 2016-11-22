#!/bin/sh
killall xterm
go build -o clientbayes ./client/client.go
go build -o serverbayes ./server/server.go
xterm -e ./serverbayes 127.0.0.1:2600 s1&
sleep 2
xterm -e ./clientbayes hopital1 127.0.0.1:2601 127.0.0.1:2600 testdata/x1.txt testdata/y1.txt h1 &
xterm -e ./clientbayes hopital2 127.0.0.1:2602 127.0.0.1:2600 testdata/x2.txt testdata/y2.txt h2 &
xterm -e ./clientbayes hopital3 127.0.0.1:2603 127.0.0.1:2600 testdata/x3.txt testdata/y3.txt h3 &
