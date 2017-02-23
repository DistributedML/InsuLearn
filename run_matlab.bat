rem echo off
set PATH=C:\Program Files\Matlab\R2016a\bin\win64;C:\TDM-GCC-64\bin;%PATH%
rem cd %GOPATH%
go build -o clientbayes.exe .\clientMatlab\client_azure.go
go build -o serverbayes.exe .\serverMatlab\server_azure.go
pause
start cmd /k .\serverbayes.exe 127.0.0.1:2600 s1
timeout 5
start cmd /k .\clientbayes.exe hospital1 127.0.0.1:2601 127.0.0.1:2600 %GOPATH%\src\github.com\4180122\distbayes\testdata\x1.txt %GOPATH%\src\github.com\4180122\distbayes\testdata\y1.txt h1 & 
start cmd /k .\clientbayes.exe hospital2 127.0.0.1:2602 127.0.0.1:2600 %GOPATH%\src\github.com\4180122\distbayes\testdata\x1.txt %GOPATH%\src\github.com\4180122\distbayes\testdata\y1.txt h2 & 
start cmd /k .\clientbayes.exe hospital3 127.0.0.1:2603 127.0.0.1:2600 %GOPATH%\src\github.com\4180122\distbayes\testdata\x1.txt %GOPATH%\src\github.com\4180122\distbayes\testdata\y1.txt h3 &
start cmd /k .\clientbayes.exe hospital4 127.0.0.1:2604 127.0.0.1:2600 %GOPATH%\src\github.com\4180122\distbayes\testdata\x1.txt %GOPATH%\src\github.com\4180122\distbayes\testdata\y1.txt h1 & 
start cmd /k .\clientbayes.exe hospital5 127.0.0.1:2605 127.0.0.1:2600 %GOPATH%\src\github.com\4180122\distbayes\testdata\x1.txt %GOPATH%\src\github.com\4180122\distbayes\testdata\y1.txt h2 & 
start cmd /k .\clientbayes.exe hospital6 127.0.0.1:2606 127.0.0.1:2600 %GOPATH%\src\github.com\4180122\distbayes\testdata\x1.txt %GOPATH%\src\github.com\4180122\distbayes\testdata\y1.txt h3 &
rem start cmd /k .\clientbayes.exe hospital4 127.0.0.1:2604 127.0.0.1:2600 %GOPATH%\src\github.com\4180122\distbayes\testdata\x4.txt %GOPATH%\src\github.com\4180122\distbayes\testdata\y4.txt h4 &
rem start cmd /k .\clientbayes.exe hospital5 127.0.0.1:2605 127.0.0.1:2600 %GOPATH%\src\github.com\4180122\distbayes\testdata\x5.txt %GOPATH%\src\github.com\4180122\distbayes\testdata\y5.txt h5 &
