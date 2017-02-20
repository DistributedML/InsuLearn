rem echo off
set PATH=C:\Program Files\Matlab\R2016b\bin\win64;C:\TDM-GCC-64\bin;%PATH%
set DBPATH=%GOPATH%\src\github.com\4180122\distbayes
set NAME=hostname
set SERVERADD=insulearns0.canadacentral.cloudapp.azure.com:12456
for /f "usebackq tokens=2 delims=:" %%f in (`ipconfig ^| findstr /c:"IPv4 Address"`) do set LOCALADD=%%f:12456
set XDATA=%DBPATH%\testdata\x1.txt
set YDATA=%DBPARG%\testdata\y1.txt

go build -o %DBPATH%\clientbayes.exe %DBPATH%\clientMatlab\client_azure.go
start cmd /k %DBPATH%\clientbayes.exe %NAME% %LOCALADD% %SERVERADD% %XDATA% %YDATA% log
