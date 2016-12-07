set GOPATH=C:\work
go build -o clientbayes.exe .\client_raft\client.go
go build -o serverbayes.exe .\server_raft\server.go
pause
start cmd /c .\serverbayes.exe 127.0.0.1:2600 raftlist.txt 1 s1
timeout 3
start cmd /c .\serverbayes.exe 127.0.0.1:2601 raftlist.txt 2 s2
start cmd /c .\serverbayes.exe 127.0.0.1:2602 raftlist.txt 3 s3
start cmd /c .\serverbayes.exe 127.0.0.1:2603 raftlist.txt 4 s4
start cmd /c .\serverbayes.exe 127.0.0.1:2604 raftlist.txt 5 s5
rem timeout 5
pause
start cmd /c .\clientbayes.exe hospital1 127.0.0.1:2605 serverlist.txt testdata\x1.txt testdata\y1.txt h1 &
start cmd /c .\clientbayes.exe hospital2 127.0.0.1:2606 serverlist.txt testdata\x2.txt testdata\y2.txt h2 &
start cmd /c .\clientbayes.exe hospital3 127.0.0.1:2607 serverlist.txt testdata\x3.txt testdata\y3.txt h3 &
rem start cmd /c .\clientbayes.exe hospital4 127.0.0.1:2608 serverlist.txt testdata\x4.txt testdata\y4.txt h4 &
rem start cmd /c .\clientbayes.exe hospital5 127.0.0.1:2609 serverlist.txt testdata\x5.txt testdata\y5.txt h5 &
