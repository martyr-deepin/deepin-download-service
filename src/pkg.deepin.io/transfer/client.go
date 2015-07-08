package transfer

import (
	"errors"
	"strings"
)

const (
	DefaultFileMode = 0644
)

type StatusCheckCallback func() int32
type ProgressCallback func(int64, int64, int64)

type Request interface {
	QuerySize() (int64, error)
	DownloadRange(int64, int64) ([]byte, error)
	Download(string) error
	ConnectStatusCheck(StatusCheckCallback)
	ConnectProgress(ProgressCallback)
}

type RequestBase struct {
	statusCheck StatusCheckCallback
	progress    ProgressCallback
}

func (rb *RequestBase) ConnectStatusCheck(cbfunc StatusCheckCallback) {
	rb.statusCheck = cbfunc
}

func (rb *RequestBase) ConnectProgress(cbfunc ProgressCallback) {
	rb.progress = cbfunc
}

type Client interface {
	SupportRange() bool
	QuerySize(url string) (int64, error)
	NewRequest(url string) (Request, error)
}

func GetClient(url string) (Client, error) {
	if strings.Contains(url, "http://") {
		return GetHttpClient(url)
	}

	if strings.Contains(url, "ftp://") {
		// TODO: get the username and password by url
		addr := strings.Split(url[len("ftp://"):len(url)], "/")[0]
		client, err := GetFtpClient("anonymous", "", addr)
		return client, err
	}
	return nil, errors.New("Unknow Portoal")
}
