@echo off
Title build...
Color 0A

REM 声明采用UTF-8编码
chcp 65001


:: 2.rsrc
:2
echo "需要先安装工具命令rsrc 命令: go get github.com/akavel/rsrc"

echo rsrc manifesting...
rsrc -arch amd64 -manifest facelock.exe.manifest -o facelock.exe.syso -ico icon.ico

echo building...
go build -o facelock.exe -ldflags="-w -s  -H windowsgui"
goto end

:end
echo complete
pause