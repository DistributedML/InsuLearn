go build -o client.exe .\client\client_go_raft.go
go build -o server.exe .\server\server_go_raft.go
pause
start cmd /K .\server.exe 127.0.0.1:2600 raftlist.txt 1 s1
timeout 3
start cmd /K .\server.exe 127.0.0.1:2601 raftlist.txt 2 s2
start cmd /K .\server.exe 127.0.0.1:2602 raftlist.txt 3 s3
start cmd /K .\server.exe 127.0.0.1:2603 raftlist.txt 4 s4
start cmd /K .\server.exe 127.0.0.1:2604 raftlist.txt 5 s5
pause
start cmd /K .\client.exe hospital1 127.0.0.1:2605 serverlist.txt testdata\x1.txt testdata\y1.txt testdata\xv.txt testdata\yv.txt h1 &
start cmd /K .\client.exe hospital2 127.0.0.1:2606 serverlist.txt testdata\x2.txt testdata\y2.txt testdata\xv.txt testdata\yv.txt h2 &
start cmd /K .\client.exe hospital3 127.0.0.1:2607 serverlist.txt testdata\x3.txt testdata\y3.txt testdata\xv.txt testdata\yv.txt h3 &
start cmd /c .\client.exe hospital4 127.0.0.1:2608 serverlist.txt testdata\x4.txt testdata\y4.txt testdata\xv.txt testdata\yv.txt h4 &
start cmd /c .\client.exe hospital5 127.0.0.1:2609 serverlist.txt testdata\x5.txt testdata\y5.txt testdata\xv.txt testdata\yv.txt h5 &
