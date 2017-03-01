:: Update according to install
set MATLABBIN=C:\Program Files\Matlab\R2016b\bin\win64
set GCCPATH=C:\TDM-GCC-64\bin

set PATH=%MATLABBIN%;%GCCPATH%;%PATH%
set DBPATH=%GOPATH%\src\github.com\4180122\distbayes

set SERVERADD=%DBPATH%\serverlist-azure.txt

for /f "usebackq tokens=2 delims=:" %%f in (`ipconfig ^| findstr /c:"IPv4 Address"`) do set LOCALADD=%%f
set XDATA=%DBPATH%\testdata\x1.txt
set YDATA=%DBPATH%\testdata\y1.txt

go build -o %DBPATH%\client.exe %DBPATH%\client\client_matlab_raft_azure.go

%DBPATH%\client.exe %LOCALADD% %LOCALADD%:12456 %SERVERADD% %XDATA% %YDATA% log 0
:: Uncomment the lines below to allow the clients to restart
rem :loop
rem timeout 10
rem %DBPATH%\client.exe %LOCALADD% %LOCALADD%:12456 %SERVERADD% %XDATA% %YDATA% log 1
rem goto loop
