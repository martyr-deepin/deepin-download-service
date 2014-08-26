package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type HttpClient struct {
	client http.Client
}

type HttpRequest struct {
	url    string
	client *HttpClient
	RequestBase
}

func (p *HttpRequest) QuerySize() (int64, error) {
	return p.client.QuerySize(p.url)
}

func (hr *HttpRequest) DownloadRange(begin int64, end int64) ([]byte, error) {
	reqest, _ := http.NewRequest("GET", hr.url, nil)
	bytestr := "bytes=" + strconv.Itoa(int(begin)) + "-" + strconv.Itoa(int(end))
	logger.Infof("[DownloadRange] %v", bytestr)
	reqest.Header.Set("Range", bytestr)
	response, err := hr.client.client.Do(reqest)
	retryTimes := 3
	for 0 < retryTimes {
		retryTimes--
		if nil != err {
			logger.Warningf("[DownloadRange] Retry")
			time.Sleep(100 * time.Millisecond)
			response, err = hr.client.client.Do(reqest)
		} else {
			break
		}
	}

	if (nil == response) || (nil != err) {
		logger.Errorf("[DownloadRange] Get Http Respone %v Failed: %v", response, err)
		return nil, err
	}

	if (response.StatusCode == 200) || (response.StatusCode == 206) {
		capacity := end - begin + 512
		buf := make([]byte, 0, capacity)
		for {
			if TASK_ST_CANCEL == hr.statusCheck() {
				return buf, TransferError("Download Cancel")
			}
			m, e := response.Body.Read(buf[len(buf):cap(buf)])
			buf = buf[0 : len(buf)+m]
			if nil != hr.progress {
				hr.progress(int64(m), begin+int64(len(buf)), 0)
			}
			if e == io.EOF {
				break
			}
			if e != nil {
				time.Sleep(500 * time.Millisecond)
				logger.Info("Read e: ", e)
				return buf, e
			}
		}
		return buf, nil
	}

	return nil, TransferError(fmt.Sprintf("[DownloadRange] Error Respone Code: %v", response.StatusCode))
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
	reqest, _ := http.NewRequest("GET", url, nil)
	fileSize := int64(0)
	response, _ := p.client.Do(reqest)
	if nil == response {
		return fileSize, TransferError("Http Request Error, Url: " + url)
	}

	if response.StatusCode == 200 {
		fileSizeStr := string(response.Header.Get("Content-Length"))
		logger.Warningf("Remote File Size: %v", fileSizeStr)
		size, err := strconv.Atoi(fileSizeStr)
		if err != nil {
			logger.Error("Set file Size")
			fileSize = 0
		}
		fileSize = int64(size)
		if 0 == fileSize {
			logger.Warning("Maybe Server Do not support Content-Length")
		}
		return fileSize, nil
	}
	return fileSize, TransferError("Http Request Error, Url: " + url)
}
