rem echo off
set PATH=C:\Program Files\Matlab\R2016b\bin\win64;C:\TDM-GCC-64\bin;%PATH%
set DBPATH=%GOPATH%\src\github.com\4180122\distbayes
go build -o %DBPATH%\serverbayes.exe %DBPATH%\serverMatlab\server_azure.go
start cmd /k %DBPATH%\serverbayes.exe 127.0.0.1:12456 log