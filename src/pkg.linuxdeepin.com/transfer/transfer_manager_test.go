package transfer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func ProgressHandle(string, int64, int64, int64) {
}

func TestHttpDownload(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, welcome to test http server!")
	}))
	defer ts.Close()

	finishChan := make(chan int)
	service := GetTransferManager()
	service.CallBack.RegisterProcessReporter(ProgressHandle)
	service.CallBack.RegisterFinishReporter(func(id string, retCode int32) {
		switch retCode {
		case TaskSuccess:
			t.Log("Download Sucess")
		default:
			t.Error("Download Failed")
		}
		finishChan <- 0
	})
	service.Lib.Download(ts.URL, TmpDir+"/monodevelop.deb", "2ced1290dee1737c6679bc9de71b2086", OnDupOverWrite)

	<-finishChan
}

func TestFtpDownload(t *testing.T) {
	finishChan := make(chan int)
	service := GetTransferManager()
	service.CallBack.RegisterProcessReporter(ProgressHandle)
	service.CallBack.RegisterFinishReporter(func(id string, retCode int32) {
		switch retCode {
		case TaskSuccess:
			t.Log("Download Sucess")
		default:
			t.Error("Download Failed")
		}
		finishChan <- 0
	})
	service.Lib.Download("ftp://127.0.0.1:8021/public/test",
		TmpDir+"/ftp.deb",
		"77f1a79cd3b26c493eeadf834038feb0",
		OnDupOverWrite)

	<-finishChan

	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		fmt.Println("Sleep", i)
	}

	service.Lib.Download("ftp://127.0.0.1:8021/public/test",
		TmpDir+"/ftp.deb",
		"77f1a79cd3b26c493eeadf834038feb0",
		OnDupOverWrite)

	<-finishChan
}
