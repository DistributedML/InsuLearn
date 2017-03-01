#!/bin/sh
killall xterm
go build -o client ./client_raft/client.go
go build -o server ./server_raft/server.go
xterm -e ./server 127.0.0.1:2600 raftlist.txt 1 s1&
sleep 3
xterm -e ./server 127.0.0.1:2601 raftlist.txt 2 s2&
xterm -e ./server 127.0.0.1:2602 raftlist.txt 3 s3&
xterm -e ./server 127.0.0.1:2603 raftlist.txt 4 s4&
xterm -e ./server 127.0.0.1:2604 raftlist.txt 5 s5&
sleep 10
xterm -e ./client hospital1 127.0.0.1:2605 serverlist.txt testdata/x1.txt testdata/y1.txt testdata/xv.txt testdata/yv.txt h1 &
xterm -e ./client hospital2 127.0.0.1:2606 serverlist.txt testdata/x2.txt testdata/y2.txt testdata/xv.txt testdata/yv.txt h2 &
xterm -e ./client hospital3 127.0.0.1:2607 serverlist.txt testdata/x3.txt testdata/y3.txt testdata/xv.txt testdata/yv.txt h3 &
xterm -e ./client hospital4 127.0.0.1:2608 serverlist.txt testdata/x4.txt testdata/y4.txt testdata/xv.txt testdata/yv.txt h4 &
xterm -e ./client hospital5 127.0.0.1:2609 serverlist.txt testdata/x5.txt testdata/y5.txt testdata/xv.txt testdata/yv.txt h5 &
