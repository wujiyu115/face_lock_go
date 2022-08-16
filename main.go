package main

import (
	"flag"
	"image"
	"io/ioutil"
	"path/filepath"
	"syscall"
	"time"

	"github.com/getlantern/systray"
	log "github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
	"gopkg.in/yaml.v2"
)

type Cfg struct {
	LogLevel            string `yaml:"logLevel"`
	LogFileName         string `yaml:"logFileName"`
	LogFileMaxAge       int64  `yaml:"logFileMaxAge"`
	LogFileRotationTime int64  `yaml:"logFileRotationTime"`
	Model               string `yaml:"model"`
	Proto               string `yaml:"proto"`
	DeviceID            int    `yaml:"deviceID"`
}

const (
	IDLE_LOCK_TIME float32 = 10
)

func checkAndLock(cfg *Cfg) {
	// parse args
	deviceID := cfg.DeviceID
	backend := gocv.NetBackendDefault

	target := gocv.NetTargetCPU

	// open capture device
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		log.Infof("Error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	img := gocv.NewMat()
	defer img.Close()

	// open DNN object tracking model
	net := gocv.ReadNet(cfg.Model, cfg.Proto)
	if net.Empty() {
		log.Errorf("Error reading network model from : %v %v\n", cfg.Model, cfg.Proto)
		return
	}
	defer net.Close()
	net.SetPreferableBackend(gocv.NetBackendType(backend))
	net.SetPreferableTarget(gocv.NetTargetType(target))

	var ratio float64
	var mean gocv.Scalar
	var swapRGB bool

	if filepath.Ext(cfg.Model) == ".caffemodel" {
		ratio = 1.0
		mean = gocv.NewScalar(104, 177, 123, 0)
		swapRGB = false
	} else {
		ratio = 1.0 / 127.5
		mean = gocv.NewScalar(127.5, 127.5, 127.5, 0)
		swapRGB = true
	}

	log.Info("Start checkAndLock")

	if ok := webcam.Read(&img); !ok {
		log.Errorf("Device closed: %v\n", deviceID)
		return
	}
	if img.Empty() {
		return
	}
	img.ConvertTo(&img, gocv.MatTypeCV32F)

	// convert image Mat to 300x300 blob that the object detector can analyze
	blob := gocv.BlobFromImage(img, ratio, image.Pt(300, 300), mean, swapRGB, false)

	// feed the blob into the detector
	net.SetInput(blob, "")

	// run a forward pass thru the network
	prob := net.Forward("")

	foundFace := false
	for i := 0; i < prob.Total(); i += 7 {
		confidence := prob.GetFloatAt(0, i+2)
		if confidence > 0.5 {
			foundFace = true
		}
	}
	prob.Close()
	blob.Close()
	if !foundFace {
		log.Info("Lock checkAndLock")
		lockWorkStation()
	}
	log.Info("End checkAndLock")
}

func onReady() {
	systray.SetIcon(iconByte)
	systray.SetTitle("face")
	systray.SetTooltip("服务已最小化右下角, 右键点击打开菜单！")
	mShow := systray.AddMenuItem("显示", "显示窗口")
	mHide := systray.AddMenuItem("隐藏", "隐藏窗口")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("退出", "退出程序")

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")
	// https://docs.microsoft.com/en-us/windows/console/getconsolewindow
	getConsoleWindows := kernel32.NewProc("GetConsoleWindow")
	// https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-showwindowasync
	showWindowAsync := user32.NewProc("ShowWindowAsync")
	consoleHandle, r2, err := getConsoleWindows.Call()
	if consoleHandle == 0 {
		log.Error("Error call GetConsoleWindow: ", consoleHandle, r2, err)
	}

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				mShow.Disable()
				mHide.Enable()
				r1, r2, err := showWindowAsync.Call(consoleHandle, 5)
				if r1 != 1 {
					log.Error("Error call ShowWindow @SW_SHOW: ", r1, r2, err)
				}
			case <-mHide.ClickedCh:
				mHide.Disable()
				mShow.Enable()
				r1, r2, err := showWindowAsync.Call(consoleHandle, 0)
				if r1 != 1 {
					log.Error("Error call ShowWindow @SW_HIDE: ", r1, r2, err)
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()

}

func onExit() {
	// clean up here
}

func main() {
	configFpath := flag.String("config", "./config.yaml", "config file")
	flag.Parse()
	config, err := ioutil.ReadFile(*configFpath)
	checkIfError(err)

	cfg := &Cfg{}
	err = yaml.Unmarshal(config, &cfg)
	checkIfError(err)

	logInit(cfg)

	log.Infof("read cfg: %+v \n", cfg)
	go func() {
		for {
			winLocked := winLocked()
			idleTime := getIdleTime()
			log.Debugf("ping winLocked:%t idleTime:%8.3f", winLocked, idleTime)
			if !winLocked && (idleTime > IDLE_LOCK_TIME) {
				checkAndLock(cfg)
			}
			time.Sleep(5 * time.Second)
		}
	}()

	systray.Run(onReady, onExit)
}
