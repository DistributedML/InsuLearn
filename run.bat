set GOPATH=C:\work
go build -o clientbayes.exe .\client\client.go
go build -o serverbayes.exe .\server\server.go
pause
start cmd /c .\serverbayes.exe 127.0.0.1:2600 s1
timeout 5
start cmd /c .\clientbayes.exe hospital1 127.0.0.1:2601 127.0.0.1:2600 testdata\x1.txt testdata\y1.txt h1 &
start cmd /c .\clientbayes.exe hospital2 127.0.0.1:2602 127.0.0.1:2600 testdata\x2.txt testdata\y2.txt h2 &
start cmd /c .\clientbayes.exe hospital3 127.0.0.1:2603 127.0.0.1:2600 testdata\x3.txt testdata\y3.txt h3 &
start cmd /c .\clientbayes.exe hospital4 127.0.0.1:2604 127.0.0.1:2600 testdata\x4.txt testdata\y4.txt h4 &
start cmd /c .\clientbayes.exe hospital5 127.0.0.1:2605 127.0.0.1:2600 testdata\x5.txt testdata\y5.txt h5 &
