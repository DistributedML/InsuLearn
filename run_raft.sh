#!/bin/sh
killall xterm
go build -o clientbayes ./client/client.go
go build -o serverbayes ./server_raft/server.go
xterm -e ./serverbayes 127.0.0.1:2600 serverlist.txt 1 s1&
sleep 3
xterm -e ./serverbayes 127.0.0.1:2601 serverlist.txt 2 s2&
xterm -e ./serverbayes 127.0.0.1:2602 serverlist.txt 3 s3&
xterm -e ./serverbayes 127.0.0.1:2603 serverlist.txt 4 s4&
xterm -e ./serverbayes 127.0.0.1:2604 serverlist.txt 5 s5&
sleep 10
xterm -e ./clientbayes hospital1 127.0.0.1:2605 127.0.0.1:2600 testdata/x1.txt testdata/y1.txt h1 &
xterm -e ./clientbayes hospital2 127.0.0.1:2606 127.0.0.1:2600 testdata/x2.txt testdata/y2.txt h2 &
xterm -e ./clientbayes hospital3 127.0.0.1:2607 127.0.0.1:2600 testdata/x3.txt testdata/y3.txt h3 &
xterm -e ./clientbayes hospital4 127.0.0.1:2608 127.0.0.1:2600 testdata/x4.txt testdata/y4.txt h4 &
xterm -e ./clientbayes hospital5 127.0.0.1:2609 127.0.0.1:2600 testdata/x5.txt testdata/y5.txt h5 &
