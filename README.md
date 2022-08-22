# 简介
此工具目的是实现离开座位自动锁屏效果

原理为: 定期查询是否有键鼠操作，超过设定时间没有输入后启动自动人脸识别, 没人后自动锁屏, 锁屏状态下也不会调用人脸识别.

# 开发
## debug
```
go build .
```
## publish
```
执行pack.bat
```

# 配置
配置文件为`yaml`格式,内容如下
```
# 日志等级 panic fatal error warn info debug trace
logLevel: "debug"
# 日志文件名
logFileName: "facelock.log"
# 日志保留天数
logFileMaxAge: 7
# 日志切割间隔(单位为小时)
logFileRotationTime: 1
# 使用第几个摄像头0开始
deviceID: 0
# 程序主循环检查间隔(秒)
checkTime: 5
# 没有键鼠操作空余时间(秒)
idleTIme: 10
# 是否记录日志到文件
islogFile: false
# 是否默认开启
isOpen: true
```

## 其他
未指定配置文件时用程序自带的默认配置运行, 你也可以指定配置运行：
```
facelock.exe --config config.yaml
```