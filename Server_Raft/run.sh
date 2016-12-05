#!/bin/sh
killall xterm
go build -o clientbayes ./client/client.go
go build -o Server1 ./Server_Raft/Server.go
cp Server1 Server2
cp Server1 Server3
cp Server1 Server4
cp Server1 Server5
# go build -o Server2 ./Server_Raft/Server.go
# go build -o Server3 ./Server_Raft/Server.go
# go build -o Server4 ./Server_Raft/Server.go
# go build -o Server5 ./Server_Raft/Server.go
xterm -e ./Server1 127.0.0.1:2600 1 server1&
xterm -e ./Server2 127.0.0.1:2601 2 server2&
xterm -e ./Server3 127.0.0.1:2602 3 server3&
xterm -e ./Server4 127.0.0.1:2603 4 server4&
xterm -e ./Server5 127.0.0.1:2604 5 server5&
sleep 2
xterm -e ./clientbayes hospital1 127.0.0.1:2601 127.0.0.1:2600 testdata/x1.txt testdata/y1.txt h1 &
xterm -e ./clientbayes hospital2 127.0.0.1:2602 127.0.0.1:2600 testdata/x2.txt testdata/y2.txt h2 &
xterm -e ./clientbayes hospital3 127.0.0.1:2603 127.0.0.1:2600 testdata/x3.txt testdata/y3.txt h3 &
