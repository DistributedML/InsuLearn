go build -o client.exe .\client\client_go.go
go build -o server.exe .\server\server_go.go
pause
start cmd /K .\server.exe 127.0.0.1:2600 s1
timeout 5
start cmd /K .\client.exe hospital1 127.0.0.1:2601 127.0.0.1:2600 testdata\x1.txt testdata\y1.txt testdata\xv.txt testdata\yv.txt h1 &
start cmd /K .\client.exe hospital2 127.0.0.1:2602 127.0.0.1:2600 testdata\x2.txt testdata\y2.txt testdata\xv.txt testdata\yv.txt h2 &
start cmd /K .\client.exe hospital3 127.0.0.1:2603 127.0.0.1:2600 testdata\x3.txt testdata\y3.txt testdata\xv.txt testdata\yv.txt h3 &
start cmd /K .\client.exe hospital4 127.0.0.1:2604 127.0.0.1:2600 testdata\x4.txt testdata\y4.txt testdata\xv.txt testdata\yv.txt h4 &
start cmd /K .\client.exe hospital5 127.0.0.1:2605 127.0.0.1:2600 testdata\x5.txt testdata\y5.txt testdata\xv.txt testdata\yv.txt h5 &
