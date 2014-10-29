package transfer

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	//Quick retry of http
	HttpRetryTimes = 3
)

type HttpClient struct {
	client http.Client
}

var _httpClientPool map[string](*HttpClient)

func init() {
	_httpClientPool = map[string](*HttpClient){}
}

func GetHttpClient(url string) (*HttpClient, error) {
	host := strings.Split(strings.Replace(url, "http://", "", -1), "/")[0]
	client := _httpClientPool[host]
	if nil == client {
		client = &HttpClient{}
		client.client = http.Client{}
	}
	return client, nil
}

type HttpRequest struct {
	url    string
	client *HttpClient
	RequestBase
}

func (hr *HttpRequest) QuerySize() (int64, error) {
	return hr.client.QuerySize(hr.url)
}

func (hr *HttpRequest) Download(localFilePath string) error {
	logger.Infof("Download %v", localFilePath)
	var err error
	dlfile, err := os.OpenFile(localFilePath, os.O_CREATE|os.O_RDWR, DefaultFileMode)
	defer dlfile.Close()
	if err != nil {
		logger.Errorf("OpenFile %v Failed: %v", localFilePath, err)
		return err
	}

	request, err := http.NewRequest("GET", hr.url, nil)
	if nil != err {
		logger.Errorf("Download Failed:\n\tRequest: %v\n\tUrl: %v\n\tError: %v", request, hr.url, err)
	}

	response, err := hr.client.client.Do(request)
	if (nil == response) || (nil != err) {
		logger.Errorf("Download Failed:\n\tRespone: %v\n\tUrl: %v\n\tError: %v", response, hr.url, err)
		return err
	}

	if response.StatusCode == 200 {
		capacity := CacheSize
		writtenBytes := int64(0)
		buf := make([]byte, 0, capacity)
		for {
			if TaskCancel == hr.statusCheck() {
				return fmt.Errorf("Cancel Download " + hr.url)
			}
			m, e := response.Body.Read(buf[len(buf):cap(buf)])
			buf = buf[0 : len(buf)+m]

			if nil != hr.progress {
				hr.progress(int64(m), writtenBytes+int64(len(buf)), 0)
			}

			if len(buf) == cap(buf) {
				dlfile.WriteAt(buf, writtenBytes)
				dlfile.Sync()
				writtenBytes += int64(len(buf))
				buf = make([]byte, 0, capacity)
				continue
			}

			if e == io.EOF {
				logger.Info("Download Read Buffer End with", e)
				break
			}
			if nil != e {
				logger.Errorf("Download %v Error: %v", hr.url, e)
				time.Sleep(ErrorRetryWaitTime * time.Millisecond)
				return e
			}
		}
	}
	return fmt.Errorf("Range Download Failed:\n\tUrl: %v\n\tStatusCode: %v\n\tStatus: %v",
		hr.url, response.StatusCode, response.Status)
}

func (hr *HttpRequest) DownloadRange(begin int64, end int64) ([]byte, error) {
	request, err := http.NewRequest("GET", hr.url, nil)
	if nil != err {
		logger.Errorf("Download Failed:\n\tRequest: %v\n\tUrl: %v\n\tError: %v", request, hr.url, err)
		return nil, err
	}

	bytestr := "bytes=" + strconv.Itoa(int(begin)) + "-" + strconv.Itoa(int(end))
	logger.Infof("Download Range %v", bytestr)
	request.Header.Set("Range", bytestr)

	response, err := hr.client.client.Do(request)

	if nil != err {
		logger.Errorf("Range Download Failed:\n\tRespone: %v\n\tUrl: %v\n\tError: %v", response, hr.url, err)
		return nil, err
	}

	if nil == response {
		logger.Errorf("Range Download Failed:\n\tRespone: %v\n\tUrl: %v\n\tError: %v", response, hr.url, err)
		return nil, fmt.Errorf("Empty respone of %v", hr.url)
	}

	if (response.StatusCode == 200) || (response.StatusCode == 206) {
		capacity := end - begin + 512
		buf := make([]byte, 0, capacity)
		for {
			if TaskCancel == hr.statusCheck() {
				return buf, fmt.Errorf("Cancel Range Download" + hr.url)
			}
			m, e := response.Body.Read(buf[len(buf):cap(buf)])
			buf = buf[0 : len(buf)+m]
			if nil != hr.progress {
				hr.progress(int64(m), begin+int64(len(buf)), 0)
			}
			if e == io.EOF {
				logger.Warningf("Read response %v Finish: %v", hr.url, e)
				break
			}
			if e != nil {
				time.Sleep(500 * time.Millisecond)
				logger.Errorf("Read response %v failed: %v", hr.url, e)
				return buf, e
			}
		}
		return buf, nil
	}

	return nil, fmt.Errorf("Range Download Failed:\n\tUrl: %v\n\tStatusCode: %v\n\tStatus: %v",
		hr.url, response.StatusCode, response.Status)
}

func (p *HttpClient) NewRequest(url string) (Request, error) {
	request := &HttpRequest{}
	request.url = url
	request.client = p
	return request, nil
}

func (p *HttpClient) SupportRange() bool {
	return true
}

func (p *HttpClient) QuerySize(url string) (int64, error) {
	fileSize := int64(0)
	reqest, err := http.NewRequest("GET", url, nil)
	if nil != err {
		return fileSize, err
	}
	response, err := p.client.Do(reqest)

	if (nil == response) || (nil != err) {
		return fileSize, fmt.Errorf("Get FileSize Failed:\n\tUrl: %v\n\tRespone: %v\n\tError: %v",
			url, response, err)
	}

	if response.StatusCode == 200 {
		fileSizeStr := string(response.Header.Get("Content-Length"))
		size, err := strconv.Atoi(fileSizeStr)
		if err != nil {
			logger.Error("Convert %v to Int Failed", fileSizeStr)
			fileSize = 0
		}
		fileSize = int64(size)
		if 0 == fileSize {
			logger.Warning("Server Do not support Content-Length")
		}
		logger.Infof("Remote File Size: %v %v", fileSizeStr, fileSize)
		return fileSize, nil
	}
	return fileSize, fmt.Errorf("Get File Size Failed:\n\tUrl: %v\n\tStatusCode: %v\n\tStatus: %v",
		url, response.StatusCode, response.Status)
}
