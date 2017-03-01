:: Parsing azure log output to measure run-time from initial to final commit
echo off
del processed-Log.txt
findstr "commit" "log-Log.txt" >> "processed-Log.txt"
rem findstr /n . "processed-Log.txt" | findstr ^1:
set /p firstline=< "processed-Log.txt"
for /f "delims==" %%a in (processed-Log.txt) do set lastline=%%a
echo %firstline%
echo %lastline%
pause
