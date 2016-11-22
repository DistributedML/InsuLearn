#!/bin/sh
killall xterm
go build github.com/4180122/distbayes/client/client.go
go build github.com/4180122/distbayes/server/server.go
xterm -e ./server 127.0.0.1:2600 &
sleep 2
xterm -e ./client hopital1 127.0.0.1:2601 127.0.0.1:2600 testdata/x1.txt testdata/y1.txt h1 &
xterm -e ./client hopital2 127.0.0.1:2602 127.0.0.1:2600 testdata/x2.txt testdata/y2.txt h2 &
xterm -e ./client hopital3 127.0.0.1:2603 127.0.0.1:2600 testdata/x3.txt testdata/y3.txt h3 &
