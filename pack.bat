@echo off
Title build...
Color 0A

REM 声明采用UTF-8编码
chcp 65001

echo rsrc manifesting...
start tools/rsrc.exe -arch amd64 -manifest tools/facelock.exe.manifest -o facelock.exe.syso -ico icon.ico

echo building...
go build -o facelock.exe -ldflags="-w -s  -H windowsgui"

echo upx.....
start tools/upx.exe facelock.exe
echo complete
pause