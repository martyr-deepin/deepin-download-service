package transfer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func ProgressHandle(string, int64, int64, int64) {
}

func TestHttpDownload(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, welcome to test http server!")
	}))
	defer ts.Close()

	finishChan := make(chan int)
	service := GetService()
	service.ProcessReport = ProgressHandle
	service.FinishReport = func(id string, retCode int32) {
		switch retCode {
		case TaskSuccess:
			t.Log("Download Sucess")
		default:
			t.Error("Download Failed")
		}
		finishChan <- 0
	}
	service.Download(ts.URL, TmpDir+"/monodevelop.deb", "2ced1290dee1737c6679bc9de71b2086", OnDupOverWrite)

	<-finishChan
}
