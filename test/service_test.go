package main

import (
	dlAPI "dbus/com/deepin/download/service"
	"fmt"
	"testing"
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
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_amd64.deb",
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_i386.deb",
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-current_4.2-1deepin2_amd64.deb",
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
		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-current_4.2-1deepin2_amd64.deb",
		"http://mirrors.aliyun.com/deepin/pool/main/d/deepin-software-center-data/deepin-software-center-data_3.0.0%2bgit20140428094643~5cd82380a4_all.deb",
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
func handleSiigleTaskFinish(taskid string) {
	fmt.Println("Test_DownloadSingleTask Finish")
	wait <- T_PASS
}

func Test_DownloadSingleTask(t *testing.T) {
	dbus := GetDBus()
	urls := []string{
		"http://mirrors.aliyun.com/deepin/pool/main/d/deepin-software-center-data/deepin-software-center-data_3.0.0+git20140428094643~5cd82380a4_all.deb",
		//		"http://mirrors.aliyun.com/deepin/pool/main/m/monodevelop-4.0/monodevelop-4.0_4.2-1deepin2_amd64.deb",
	}
	md5s := []string{}
	sizes := []int64{}
	defer dbus.ConnectUpdate(handleUpdete)()
	defer dbus.ConnectFinish(handleSiigleTaskFinish)()
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
