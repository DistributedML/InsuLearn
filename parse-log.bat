echo off
del processed-Log.txt
findstr "commit" "log-Log.txt" >> "processed-Log.txt"
rem findstr /n . "processed-Log.txt" | findstr ^1:
set /p firstline=< "processed-Log.txt"
for /f "delims==" %%a in (processed-Log.txt) do set lastline=%%a
pause
for /f "tokens=2" %%a in (%firstline%) do set firsttime=%%a
rem for /f "tokens=1" %%a in (%lastline) do set lasttime=%%a
echo %firsttime%
echo %lasttime%
pause
