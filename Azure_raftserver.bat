rem echo off
set PATH=C:\Program Files\Matlab\R2016b\bin\win64;C:\TDM-GCC-64\bin;%PATH%
set DBPATH=%GOPATH%\src\github.com\4180122\distbayes
go build -o %DBPATH%\serverbayes.exe %DBPATH%\serverMatlab_raft\serverM_raft.go
%DBPATH%\serverbayes.exe :12456 %DBPATH%\raftlist-azure.txt 1 log
pause
