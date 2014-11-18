package service

import (
	"fmt"
	"testing"
)

const (
	TmpDir = "../../../misc/test/tmp"
)

const (
	T_PASS   = int32(0)
	T_FAILED = int32(1)
)

var wait chan int32

func handleUpdete(taskid string, progress int32, speed int32, finish int32, total int32, downloadBytes int64, totalBytes int64) {
	fmt.Printf("%40v %4v %12v %4v %4v %12v %12v\n", taskid, progress, speed, finish, total, downloadBytes, totalBytes)
}

var _mutiTotal = 11
var _mutiDownload = 0

func handleMutiTaskFinish(taskid string) {
	_mutiDownload++
	fmt.Printf("Task %v Finish, %v/%v\n", taskid, _mutiDownload, _mutiTotal)
	if _mutiDownload >= _mutiTotal {
		fmt.Println("MutiTask Test Pass")
		wait <- T_PASS
		return
	}
	if (_mutiStop + _mutiDownload) >= _mutiTotal {
		fmt.Println("MutiTask Test Failed")
		wait <- T_FAILED
	}
}

var _mutiStop = 0

func handleMutiTaskStop(taskid string) {
	fmt.Println("Stop Task ", taskid)
	_mutiStop++
	if (_mutiStop + _mutiDownload) >= _mutiTotal {
		fmt.Println("Stop Task ", taskid)
		wait <- T_FAILED
	}
}

func errorHandle(taskid string, errCode int32, errStr string) {
	fmt.Println("Error", taskid, errCode, errStr)
	wait <- T_FAILED
}

func waitHandle(taskid string) {
}

func waitTaskFinish(t *testing.T, excepet int32) {
	for {
		select {
		case ret := <-wait:
			if excepet == ret {
				t.Logf("Test Pass")
				return
			} else {
				t.Errorf("Test Failed")
				return
			}
		}
	}
}

func TestDownloadMutiErrorTask(t *testing.T) {
	wait = make(chan int32, 1024)
	service := GetService()

	//This url all not exist to test error sence.
	urls := []string{
		"http://errorurl.error.linuxdeepin/app1.deb",
		"http://errorurl.error.linuxdeepin/app2.deb",
		"http://errorurl.error.linuxdeepin/app3.deb",
		"http://errorurl.error.linuxdeepin/app4.deb",
	}
	md5s := []string{
		"",
	}
	sizes := []int64{
		0,
	}

	_mutiTotal = 1
	service.cbUpdate = handleUpdete
	service.cbStart = waitHandle
	service.cbFinish = handleMutiTaskFinish
	service.cbStop = handleMutiTaskStop
	service.cbError = errorHandle

	store := TmpDir
	taskid := service.addTask("moon", urls, sizes, md5s, store)
	if "" == taskid {
		t.Error("add task failed")
	}
	t.Log(taskid)
	waitTaskFinish(t, T_FAILED)
}
