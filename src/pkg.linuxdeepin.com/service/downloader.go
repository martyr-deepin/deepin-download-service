package main

import (
	"errors"
	"fmt"
	"strings"

	"pkg.linuxdeepin.com/lib/utils"
)

const (
	DownloaderWait   = int32(0xA0)
	DownloaderStart  = int32(0xA1)
	DownloaderFinish = int32(0xA2)
	DownloaderCancel = int32(0xA3)
)

type Downloader struct {
	ID           string
	status       int32
	transferID   string
	fileName     string
	storeDir     string
	url          string
	totalSize    int64
	downloadSize int64
	md5          string

	refTasks map[string](*Task)

	cacheKey   string
	chanCancel chan int32
}

const (
	C_INVAILD_TRANSFERID = ""
)

func GetUrlFileName(url string) string {
	list := strings.Split(url, "/")
	return list[len(list)-1]
}

func downloadID() string {
	return utils.GenUuid() + "_download"
}

func newDownloader(url string, totalSize int64, md5 string, storeDir string, fileName string) *Downloader {
	downloader := &Downloader{}
	downloader.ID = downloadID()
	downloader.fileName = fileName
	downloader.refTasks = map[string](*Task){}
	downloader.status = DownloaderWait
	if 0 == len(fileName) {
		downloader.fileName = GetUrlFileName(url)
	}
	downloader.transferID = C_INVAILD_TRANSFERID
	downloader.storeDir = storeDir
	downloader.url = url
	downloader.totalSize = totalSize
	downloader.md5 = md5
	return downloader
}

var _downloaderCache map[string](*Downloader)

func GetDownloader(url string, totalSize int64, md5 string, storeDir string, fileName string) *Downloader {
	if nil == _downloaderCache {
		_downloaderCache = map[string](*Downloader){}
	}
	key := url + storeDir
	dl := _downloaderCache[key]
	if nil == dl {
		dl = newDownloader(url, totalSize, md5, storeDir, fileName)
		dl.cacheKey = key
		_downloaderCache[key] = dl
	}
	return dl
}

var _downloaderIndex map[string](*Downloader)

func IndexDownloader(transferID string, dl *Downloader) {
	if nil == _downloaderIndex {
		_downloaderIndex = map[string](*Downloader){}
	}
	_downloaderIndex[transferID] = dl
}

func QueryDownloader(transferID string) *Downloader {
	if nil == _downloaderIndex {
		_downloaderIndex = map[string](*Downloader){}
	}
	return _downloaderIndex[transferID]
}

func removeDownloader(dl *Downloader) {
	_downloaderCache[dl.cacheKey] = nil
	_downloaderIndex[dl.transferID] = nil
	delete(_downloaderCache, dl.cacheKey)
	delete(_downloaderIndex, dl.transferID)
}

func DownloadError(errStr string) error {
	logger.Error(errStr)
	return errors.New(errStr)
}

func (p *Downloader) QuerySize() int64 {
	size, err := TransferDbus().QuerySize(p.url)
	if nil != err {
		logger.Error("%v QuerySize of %v error", p.ID, p.url)
	}
	p.totalSize = size
	return size
}

func (p *Downloader) Start() error {
	if p.transferID != C_INVAILD_TRANSFERID {
		return nil
	}

	result, transferID, err := TransferDbus().Download(p.url, p.storeDir+"/"+p.fileName, p.md5, 0)
	if (nil != err) || (result != 0) {
		ret := fmt.Sprintf("Start Transfer Error: Result: %v Error: %v", result, err)
		return DownloadError(ret)
	}
	p.transferID = transferID
	IndexDownloader(transferID, p)
	p.status = DownloaderStart
	return nil
}

func (p *Downloader) Pause() error {
	result, err := TransferDbus().Pause(p.transferID)
	if (0 != result) || (nil != err) {
		ret := fmt.Sprintf("Puase Downloader %v Failed", p.transferID)
		return DownloadError(ret)
	}
	return nil
}

func (p *Downloader) Resume() error {
	result, err := TransferDbus().Resume(p.transferID)
	if (0 != result) || (nil != err) {
		ret := fmt.Sprintf("Resume Downloader %v Failed", p.transferID)
		return DownloadError(ret)
	}
	return nil
}

func (p *Downloader) Cancel() error {
	result, err := TransferDbus().Cancel(p.transferID)
	if (0 != result) || (nil != err) {
		ret := fmt.Sprintf("Cancel Downloader %v Failed", p.transferID)
		return DownloadError(ret)
	}
	p.status = DownloaderCancel
	p.transferID = ""
	return nil
}

func (p *Downloader) Finish() error {
	p.refTasks = map[string](*Task){}
	p.status = DownloaderFinish
	p.transferID = ""
	removeDownloader(p)
	return nil
}

func (p *Downloader) RefTask(task *Task) error {
	p.refTasks[task.ID] = task
	return nil
}

func (p *Downloader) UnRefTask(task *Task) error {
	delete(p.refTasks, task.ID)
	if 0 == len(p.refTasks) {
		removeDownloader(p)
		go p.Cancel()
	}
	return nil
}
