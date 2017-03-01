:: This file needs to be changed on every single Raft machine. In line:
::    %DBPATH%\serverbayes.exe :12456 %DBPATH%\raftlist-azure.txt <server-ID> log
:: the <server-ID> should be set according to the server IP in the raftlist-azure.txt

:: Update according to install
set MATLABBIN=C:\Program Files\Matlab\R2016b\bin\win64
set GCCPATH=C:\TDM-GCC-64\bin

set PATH=%MATLABBIN%;%GCCPATH%;%PATH%
set DBPATH=%GOPATH%\src\github.com\4180122\distbayes

go build -o %DBPATH%\server.exe %DBPATH%\server\server_matlab_azure.go
%DBPATH%\server.exe :12456 log
pause
