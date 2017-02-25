rem echo off
set PATH=C:\Program Files\Matlab\R2016b\bin\win64;C:\TDM-GCC-64\bin;%PATH%
set DBPATH=%GOPATH%\src\github.com\4180122\distbayes
set SERVERADD=10.1.0.4:12456
rem set SERVERADD=%DBPATH%\serverlist-azure.txt
for /f "usebackq tokens=2 delims=:" %%f in (`ipconfig ^| findstr /c:"IPv4 Address"`) do set LOCALADD=%%f
set XDATA=%DBPATH%\testdata\x1.txt
set YDATA=%DBPATH%\testdata\y1.txt

rem go build -o %DBPATH%\clientbayes.exe %DBPATH%\clientMatlab_raft\clientM_raft.go
go build -o %DBPATH%\clientbayes.exe %DBPATH%\clientMatlab\client_azure.go

%DBPATH%\clientbayes.exe %LOCALADD% %LOCALADD%:12456 %SERVERADD% %XDATA% %YDATA% log 0
:loop
timeout 10
%DBPATH%\clientbayes.exe %LOCALADD% %LOCALADD%:12456 %SERVERADD% %XDATA% %YDATA% log 1

rem %DBPATH%\clientbayes.exe %LOCALADD% %LOCALADD%:12456 %SERVERADD% %XDATA% %YDATA% log 0
rem :loop
rem timeout 10
rem %DBPATH%\clientbayes.exe %LOCALADD% %LOCALADD%:12456 %SERVERADD% %XDATA% %YDATA% log 1
rem goto loop
