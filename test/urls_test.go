package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

var _urltotal = 0
var _urlsdownloads = 0
var _urlstop = 0

func handleDownUrlsFinish(taskid string) {
	_urlsdownloads++
	fmt.Printf("Task %v Finish, %v/%v\n", taskid, _urlsdownloads, _urltotal)
	if _urlsdownloads >= _urltotal {
		fmt.Println("Download Urls Test Pass")
		wait <- T_PASS
		return
	}
	if (_urlstop + _urlsdownloads) >= _urltotal {
		fmt.Println("Test Failed")
		wait <- T_FAILED
	}
}

//TestUrksDownload will get url from file and downloads
func TestUrlsDownload(t *testing.T) {
	wait = make(chan int32, 1024)
	dbus := GetDBus()

	urlsFile := "download_urls"
	buf, err := ioutil.ReadFile(urlsFile)
	if nil != err {
		t.Errorf("Open File Failed", err)
	}

	urllist := strings.Split(string(buf), "\n")

	urls := []string{}
	for _, url := range urllist {
		if len(url) > len("http://") {
			urls = append(urls, url)
		}
	}

	fmt.Println(urls)

	_urltotal = len(urls)

	md5s := []string{
		"",
	}
	sizes := []int64{
		0,
	}

	defer dbus.ConnectUpdate(handleUpdete)()
	defer dbus.ConnectFinish(handleDownUrlsFinish)()
	defer dbus.ConnectStop(handleSingleTaskStop)()

	store := "/tmp"
	taskid, err := GetDBus().AddTask("urlsdownload", urls, sizes, md5s, store)
	if nil != err {
		t.Error(err)
	}
	t.Log(taskid)
	waitTaskFinish(t)
}
