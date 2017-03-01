#!/bin/sh
killall xterm
go build -o client ./client/client_go.go
go build -o server ./server/server_go.go
xterm -e ./server 127.0.0.1:2600 s1&
sleep 2
xterm -e ./client hospital1 127.0.0.1:2601 127.0.0.1:2600 testdata/x1.txt testdata/y1.txt testdata/xv.txt testdata/yv.txt h1 &
xterm -e ./client hospital2 127.0.0.1:2602 127.0.0.1:2600 testdata/x2.txt testdata/y2.txt testdata/xv.txt testdata/yv.txt h2 &
xterm -e ./client hospital3 127.0.0.1:2603 127.0.0.1:2600 testdata/x3.txt testdata/y3.txt testdata/xv.txt testdata/yv.txt h3 &
