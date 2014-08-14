package main

import (
	dlAPI "dbus/com/deepin/download/service"
	"fmt"
	"testing"
	"time"
)

const (
	SERVICE_NAME = "com.deepin.download.service"
	SERVICE_PATH = "/com/deepin/download/service"
)

const (
	T_PASS   = int32(0)
	T_FAILED = int32(1)
)

var _dlDBus *dlAPI.Service

var wait chan int32

func GetDBus() *dlAPI.Service {
	if nil == _dlDBus {
		wait = make(chan int32)
		pauseChan = make(chan int32)
		resumeChan = make(chan int32)
		var err error
		_dlDBus, err = dlAPI.NewService(SERVICE_NAME, SERVICE_PATH)
		if nil != err {
			panic("InitDbus Error")
		}
	}
	return _dlDBus
}

func handleUpdete(taskid string, progress int32, speed int32, finish int32, total int32, downloadBytes int64, totalBytes int64) {
	fmt.Printf("%v  \t%v  \t%v  \t%v  \t%v  \t%v  \t%v\n", taskid, progress, speed, finish, total, downloadBytes, totalBytes)
}

var mutitotal = 2
var mutidownload = 0

func handleMutiTaskFinish(taskid string) {
	mutidownload++
	fmt.Printf("[handleMutiTaskFinish] Task %v Finish, %v/%v", taskid, mutidownload, mutitotal)
	if mutidownload >= mutitotal {
		fmt.Println("MutiTask Test Pass")
		wait <- T_PASS
		return
	}
	if (mutistop + mutidownload) >= mutitotal {
		fmt.Println("MutiTask Test Failed")
		wait <- T_FAILED
	}
}

var mutistop = 0

func handleMutiTaskStop(taskid string) {
	fmt.Println("Stop Task ", taskid)
	mutistop++
	if (mutistop + mutidownload) >= 2 {
		fmt.Println("Stop Task ", taskid)
		wait <- T_FAILED
	}
}

func errorHandle(taskid string, errCode int32, errStr int32) {
	fmt.Println("Error", taskid, errCode, errStr)
	wait <- T_FAILED
}

func waitTaskFinish(t *testing.T) {
	wait = make(chan int32)
	for {
		select {
		case ret := <-wait:
			switch ret {
			case T_PASS:
				t.Logf("Test Pass")
				return
			default:
				t.Errorf("Test Failed")
				return
			}
		}
	}
}

func Test_DownloadMutiTask(t *testing.T) {
	dbus := GetDBus()
	urls := []string{
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_i386.deb",
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-current_4.2-1deepin2_amd64.deb",
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_amd64.deb",
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-current_4.2-1deepin2_i386.deb",
	}
	md5s := []string{
		"",
	}
	sizes := []int64{
		0,
	}

	defer dbus.ConnectUpdate(handleUpdete)()
	defer dbus.ConnectFinish(handleMutiTaskFinish)()
	defer dbus.ConnectStop(handleMutiTaskStop)()

	store := "/tmp"
	taskid, err := GetDBus().AddTask("moon", urls, sizes, md5s, store)
	if nil != err {
		t.Error(err)
	}
	t.Log(taskid)

	urls = []string{
		"http://mirrors.aliyun.com/deepin/pool/main/d/deepin-software-center-data/deepin-software-center-data_3.0.0%2bgit20140428094643~5cd82380a4_all.deb",
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-current_4.2-1deepin2_amd64.deb",
	}

	taskid, err = GetDBus().AddTask("store", urls, sizes, md5s, store)
	if nil != err {
		t.Error(err)
	}
	t.Log(taskid)
	waitTaskFinish(t)
}

func handleSingleTaskStop(taskid string) {
	fmt.Println("Stop Task Finish", taskid)
	wait <- T_FAILED
}
func handleSigleTaskFinish(taskid string) {
	fmt.Println("Test_DownloadSingleTask Finish")
	wait <- T_PASS
}

func Test_DownloadSingleTask(t *testing.T) {
	dbus := GetDBus()
	urls := []string{
		"http://mirrors.aliyun.com/deepin/pool/main/d/deepin-software-center-data/deepin-software-center-data_3.0.0+git20140428094643~5cd82380a4_all.deb",
	}
	md5s := []string{}
	sizes := []int64{}
	defer dbus.ConnectUpdate(handleUpdete)()
	defer dbus.ConnectFinish(handleSigleTaskFinish)()
	defer dbus.ConnectStop(handleSingleTaskStop)()

	store := "/tmp"
	taskid, err := GetDBus().AddTask("moon", urls, sizes, md5s, store)
	if nil != err {
		t.Error(err)
	}
	t.Log(taskid)

	waitTaskFinish(t)
}

func handleErrorUrlFinish(taskid string) {
	fmt.Println("cb_ErrorUrl: ", taskid)
	wait <- T_FAILED
}

func handleErrorUrlStop(taskid string) {
	fmt.Println("cb_ErrorUrlStop:", taskid)
	wait <- T_PASS
}

func Test_DownloadiErrorUrl(t *testing.T) {
	dbus := GetDBus()
	urls := []string{
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_amd64.deb.eeorr",
	}
	md5s := []string{}
	sizes := []int64{}
	defer dbus.ConnectUpdate(handleUpdete)()
	defer dbus.ConnectFinish(handleErrorUrlFinish)()
	defer dbus.ConnectStop(handleErrorUrlStop)()

	store := "/tmp"
	taskid, err := GetDBus().AddTask("moon", urls, sizes, md5s, store)
	if nil != err {
		t.Error(err)
	}
	t.Log(taskid)

	waitTaskFinish(t)
}
func handleMD5Finish(taskid string) {
	fmt.Println("handleMD5Finish: ", taskid)
	wait <- T_PASS
}

func handleMD5Stop(taskid string) {
	fmt.Println("handleMD5Stop:", taskid)
	wait <- T_FAILED
}

func Test_VerifyMD5(t *testing.T) {
	dbus := GetDBus()
	urls := []string{
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_amd64.deb",
	}
	md5s := []string{"80e7028d649cb2c81fdc4eab6a94b0c7"}
	sizes := []int64{}
	defer dbus.ConnectUpdate(handleUpdete)()
	defer dbus.ConnectFinish(handleMD5Finish)()
	defer dbus.ConnectStop(handleMD5Stop)()

	store := "/tmp"
	taskid, err := GetDBus().AddTask("moon", urls, sizes, md5s, store)
	if nil != err {
		t.Error(err)
	}
	t.Log(taskid)

	waitTaskFinish(t)
}
func Test_VerifyMD5Error(t *testing.T) {
	dbus := GetDBus()
	urls := []string{
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_amd64.deb",
	}
	md5s := []string{"error80e7028d649cb2c81fdc4eab6a94b0c7"}
	sizes := []int64{}
	defer dbus.ConnectUpdate(handleUpdete)()
	defer dbus.ConnectFinish(handleMD5Stop)()
	defer dbus.ConnectStop(handleMD5Finish)()

	store := "/tmp"
	taskid, err := GetDBus().AddTask("moon", urls, sizes, md5s, store)
	if nil != err {
		t.Error(err)
	}
	t.Log(taskid)

	waitTaskFinish(t)
}

var pauseChan chan int32

func waitPause(t *testing.T) {
	for {
		select {
		case <-pauseChan:
			fmt.Print("Recive Pasue signal\n")
			return
		case <-time.After(5 * time.Second):
			t.Error("Wait Pause signal timeout")
			fmt.Print("Wait Pause signal timeout\n")
			return
		}
	}
}

var resumeChan chan int32

func waitResume(t *testing.T) {
	for {
		select {
		case <-resumeChan:
			fmt.Print("Recive Resume signal\n")
			return
		case <-time.After(5 * time.Second):
			t.Error("Wait Resume signal timeout")
			fmt.Print("Wait Resume signal timeout\n")
			return
		}
	}
}

func handleSingleTaskPause(taskid string) {
	fmt.Println("Recive Pause signal of ", taskid)
	pauseChan <- int32(1)
}

func handleSingleTaskResume(taskid string) {
	fmt.Println("Recive Resume signel of ", taskid)
	resumeChan <- int32(1)
}

func Test_PauseResume(t *testing.T) {
	dbus := GetDBus()
	urls := []string{
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_amd64.deb",
	}
	md5s := []string{"80e7028d649cb2c81fdc4eab6a94b0c7"}
	sizes := []int64{}
	defer dbus.ConnectUpdate(handleUpdete)()
	defer dbus.ConnectFinish(handleSigleTaskFinish)()
	defer dbus.ConnectStop(handleSingleTaskStop)()
	defer dbus.ConnectPause(handleSingleTaskPause)()
	defer dbus.ConnectResume(handleSingleTaskResume)()

	resumeChan = make(chan int32)
	pauseChan = make(chan int32)
	store := "/tmp"
	taskid, err := dbus.AddTask("moon", urls, sizes, md5s, store)
	if nil != err {
		t.Error(err)
	}
	t.Log(taskid)
	fmt.Print("Sleep 3s\n")
	time.Sleep(3 * time.Second)
	fmt.Print("Call PauseTask\n")
	dbus.PauseTask(taskid)

	waitPause(t)

	fmt.Print("Sleep 3s\n")
	time.Sleep(3 * time.Second)
	fmt.Print("Call ResumeTask\n")
	dbus.ResumeTask(taskid)

	waitResume(t)

	waitTaskFinish(t)
}

func handleMuti11Task(taskid string) {
	fmt.Println("Finish Task", taskid)
	wait <- T_PASS
}

func Test_DownloadMuti11Task(t *testing.T) {
	dbus := GetDBus()

	urls := []string{
		"http://packages.corp.linuxdeepin.com/hourly-build/pool/main/d/dde-control-center/dde-control-center_0.0.3+20140813104038~053bc8ae83_amd64.deb",
		"http://packages.corp.linuxdeepin.com/hourly-build/pool/main/g/go-dlib/go-dlib_0.0.4+20140813112212~527a84f2d9_amd64.deb",
		"http://packages.corp.linuxdeepin.com/hourly-build/pool/main/s/startdde/startdde_0.1+20140812172244~0dd4dac0b1_amd64.deb",
		"http://packages.corp.linuxdeepin.com/hourly-build/pool/main/d/deepin-movie/deepin-movie_0.1+20140811110444~a6058bf40f_all.deb",
		"http://packages.corp.linuxdeepin.com/hourly-build/pool/main/d/deepin-installer/deepin-installer_1.1+20140812154137~d456f5a540_amd64.deb",
		"http://packages.corp.linuxdeepin.com/hourly-build/pool/main/d/deepin-terminal/deepin-terminal_1.1+20140812184831~02e9a8a103_all.deb",
		"http://packages.corp.linuxdeepin.com/hourly-build/pool/main/d/dde-daemon/dde-daemon_0.0.1+20140813152049~c893c386a4_amd64.deb",
		"http://packages.corp.linuxdeepin.com/hourly-build/pool/main/d/deepin-qml-widgets/deepin-qml-widgets_0.0.2+20140813104315~d8f63b5560_amd64.deb",
		"http://packages.corp.linuxdeepin.com/hourly-build/pool/main/d/deepin-gtk-theme/deepin-gtk-theme_14.07+20140811102443~6ca92f7879_all.deb",
		"http://packages.corp.linuxdeepin.com/hourly-build/pool/main/d/deepin-software-center/deepin-software-center_3.0.1+20140813104534~b8346a9c54_all.deb",

		//		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_i386.deb",
		//		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-current_4.2-1deepin2_amd64.deb",
		//		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_amd64.deb",
		//		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-current_4.2-1deepin2_i386.deb",
	}
	md5s := []string{
		"",
	}
	sizes := []int64{
		0,
	}

	defer dbus.ConnectUpdate(handleUpdete)()
	defer dbus.ConnectFinish(handleMutiTaskFinish)()
	defer dbus.ConnectStop(handleSigleTaskFinish)()

	store := "/tmp"
	taskid, err := GetDBus().AddTask("moon", urls, sizes, md5s, store)
	if nil != err {
		t.Error(err)
	}
	t.Log(taskid)
	waitTaskFinish(t)
}
