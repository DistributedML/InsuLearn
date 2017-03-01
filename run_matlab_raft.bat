:: Update according to install
set MATLABBIN=C:\Program Files\Matlab\R2011b\bin\win64
set GCCPATH=C:\TDM-GCC-64\bin

set PATH=%MATLABBIN%;%GCCPATH%;%PATH%

go build -o client.exe .\client\client_matlab_raft.go
go build -o server.exe .\server\server_matlab_raft.go
pause
start cmd /K .\server.exe 127.0.0.1:2600 raftlist.txt 1 s1
timeout 3
start cmd /K .\server.exe 127.0.0.1:2601 raftlist.txt 2 s2
start cmd /K .\server.exe 127.0.0.1:2602 raftlist.txt 3 s3
start cmd /K .\server.exe 127.0.0.1:2603 raftlist.txt 4 s4
start cmd /K .\server.exe 127.0.0.1:2604 raftlist.txt 5 s5
rem timeout 5
pause
start cmd /k .\client.exe hospital1 127.0.0.1:2605 serverlist.txt %cd%\testdata\x1.txt %cd%\testdata\y1.txt %cd%\testdata\xv.txt %cd%\testdata\yv.txt h1 & 
start cmd /k .\client.exe hospital2 127.0.0.1:2606 serverlist.txt %cd%\testdata\x2.txt %cd%\testdata\y2.txt %cd%\testdata\xv.txt %cd%\testdata\yv.txt h2 & 
start cmd /k .\client.exe hospital3 127.0.0.1:2607 serverlist.txt %cd%\testdata\x3.txt %cd%\testdata\y3.txt %cd%\testdata\xv.txt %cd%\testdata\yv.txt h3 &
start cmd /k .\client.exe hospital4 127.0.0.1:2608 serverlist.txt %cd%\testdata\x4.txt %cd%\testdata\y4.txt %cd%\testdata\xv.txt %cd%\testdata\yv.txt h4 & 
start cmd /k .\client.exe hospital5 127.0.0.1:2609 serverlist.txt %cd%\testdata\x5.txt %cd%\testdata\y5.txt %cd%\testdata\xv.txt %cd%\testdata\yv.txt h5 & 
