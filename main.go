package main

import (
	"embed"
	"flag"
	"image"
	"io/ioutil"
	"syscall"
	"time"

	"github.com/getlantern/systray"
	log "github.com/sirupsen/logrus"
	"gocv.io/x/gocv"
	"gopkg.in/yaml.v2"
)

//go:embed weights
var weights embed.FS

//go:embed config.yaml
var configBytes []byte

//go:embed icon.ico
var iconByte []byte

var (
	net gocv.Net
)

type Cfg struct {
	LogLevel            string  `yaml:"logLevel"`
	LogFileName         string  `yaml:"logFileName"`
	LogFileMaxAge       int64   `yaml:"logFileMaxAge"`
	LogFileRotationTime int64   `yaml:"logFileRotationTime"`
	Model               string  `yaml:"model"`
	Proto               string  `yaml:"proto"`
	DeviceID            int     `yaml:"deviceID"`
	CheckTime           int     `yaml:"checkTime"`
	IdleTIme            float32 `yaml:"idleTIme"`
}

func checkAndLock(cfg *Cfg, n *gocv.Net) {
	log.Info("Start checkAndLock")
	// parse args
	deviceID := cfg.DeviceID

	// open capture device
	webcam, err := gocv.OpenVideoCapture(deviceID)
	if err != nil {
		log.Infof("Error opening video capture device: %v\n", deviceID)
		return
	}
	defer webcam.Close()

	img := gocv.NewMat()
	defer img.Close()

	var ratio float64 = 1.0
	var mean gocv.Scalar = gocv.NewScalar(104, 177, 123, 0)
	var swapRGB bool = false

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
	n.SetInput(blob, "")
	// run a forward pass thru the network
	prob := n.Forward("")

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
	getConsoleWindows := kernel32.NewProc("GetConsoleWindow")
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
	log.Info("onExit")
	net.Close()
}

func main() {
	configFpath := flag.String("config", "none", "config file")
	flag.Parse()

	if *configFpath != "none" {
		var err error
		configBytes, err = ioutil.ReadFile(*configFpath)
		checkIfError(err)
	}

	cfg := &Cfg{}
	err := yaml.Unmarshal(configBytes, &cfg)
	checkIfError(err)
	logInit(cfg)

	log.Infof("read cfg: %+v \n", cfg)

	proto, err := weights.ReadFile("weights/deploy.prototxt.txt")
	checkIfError(err)
	caffemodel, err := weights.ReadFile("weights/res10_300x300_ssd_iter_140000_fp16.caffemodel")
	checkIfError(err)

	// open DNN object tracking model
	net, err := gocv.ReadNetFromCaffeBytes(proto, caffemodel)
	if err != nil {
		log.Errorf("Error reading network model")
		return
	}
	net.SetPreferableBackend(gocv.NetBackendType(gocv.NetBackendDefault))
	net.SetPreferableTarget(gocv.NetTargetType(gocv.NetTargetCPU))

	go func() {
		for {
			winLocked := winLocked()
			idleTime := getIdleTime()
			log.Debugf("ping winLocked:%t idleTime:%8.3f", winLocked, idleTime)
			if !winLocked && (idleTime > cfg.IdleTIme) {
				checkAndLock(cfg, &net)
			}
			time.Sleep(time.Duration(cfg.CheckTime) * time.Second)
		}
	}()

	systray.Run(onReady, onExit)
}
