set FILE=Shiviz.log
echo ^(?^<host^>\S*^) ^(?^<clock^>{.*}^)\n(?^<event^>.*^)  > %FILE%
echo. >> %FILE%
copy /a %FILE%+*-Log.txt %FILE%
rem del -Log.txt
