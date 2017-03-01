:: Update according to install
set MATLABBIN=C:\Program Files\Matlab\R2011b\bin\win64
set GCCPATH=C:\TDM-GCC-64\bin

set PATH=%MATLABBIN%;%GCCPATH%;%PATH%

go build -o client.exe .\client\client_matlab.go
go build -o server.exe .\server\server_matlab.go
pause
start cmd /k .\server.exe 127.0.0.1:2600 s1
timeout 5
start cmd /k .\client.exe hospital1 127.0.0.1:2601 127.0.0.1:2600 %cd%\testdata\x1.txt %cd%\testdata\y1.txt %cd%\testdata\xv.txt %cd%\testdata\yv.txt h1 & 
start cmd /k .\client.exe hospital2 127.0.0.1:2602 127.0.0.1:2600 %cd%\testdata\x2.txt %cd%\testdata\y2.txt %cd%\testdata\xv.txt %cd%\testdata\yv.txt h2 & 
start cmd /k .\client.exe hospital3 127.0.0.1:2603 127.0.0.1:2600 %cd%\testdata\x3.txt %cd%\testdata\y3.txt %cd%\testdata\xv.txt %cd%\testdata\yv.txt h3 &
start cmd /k .\client.exe hospital4 127.0.0.1:2604 127.0.0.1:2600 %cd%\testdata\x4.txt %cd%\testdata\y4.txt %cd%\testdata\xv.txt %cd%\testdata\yv.txt h4 & 
start cmd /k .\client.exe hospital5 127.0.0.1:2605 127.0.0.1:2600 %cd%\testdata\x5.txt %cd%\testdata\y5.txt %cd%\testdata\xv.txt %cd%\testdata\yv.txt h5 & 
