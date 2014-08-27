package main

import "testing"

func Test_VereifyMD5P(t *testing.T) {
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

func Test_QuickDownload(t *testing.T) {
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
