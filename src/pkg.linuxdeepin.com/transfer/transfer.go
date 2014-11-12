/*
Copyright (C) 2011~2014 Deepin, Inc.
              2011~2014 He Li

Author:     He Li <me@iceyer.net>
Maintainer: He Li <me@iceyer.net>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package transfer

import (
	"container/list"
	"fmt"

	"os"
	"runtime"
	"strconv"
	"time"

	"pkg.linuxdeepin.com/lib/utils"
)

const (
	TaskStart   = int32(0x10)
	TaskSuccess = int32(0x11)
	TaskFailed  = int32(0x12)
	TaskNoExist = int32(0x13)
	TaskPause   = int32(0x14)
	TaskCancel  = int32(0x15)
)

const (
	OnDupRename    = int32(0x40)
	OnDupOverWrite = int32(0x41)
)

const (
	downloadRetryTime      = 10
	rangedownloadRetryTime = 5
)

type Transfer struct {
	ID                  string
	status              int32
	url                 string
	md5                 string
	ondup               int32
	fileSize            int64
	fileName            string
	originLocalFileName string
	localFile           string
	statusFile          string

	taskStatusChan chan int32

	element *list.Element

	detaSize     int64
	downloadSize int64
	totalSize    int64
}

func newTransferID() string {
	return utils.GenUuid() + "_transfer"
}

//NewTask
func NewTransfer(url string, localFile string, md5 string, ondup int32) (*Transfer, error) {
	t := &Transfer{}
	t.ID = newTransferID()
	t.taskStatusChan = make(chan int32)
	t.status = TaskPause
	t.md5 = md5
	t.url = url
	t.originLocalFileName = localFile
	t.localFile = localFile
	t.status = TaskStart
	t.ondup = ondup
	logger.Warningf("[NewTransfer] ID: %v\n\turl: %v", t.ID, url)
	return t, nil
}

func (t *Transfer) Download() error {
	t.status = TaskStart
	var err error
	retryTime := 0
	for retryTime < downloadRetryTime {
		retryInterval := 500 * time.Duration(retryTime*retryTime) * time.Millisecond
		time.Sleep(retryInterval)
		retryTime++
		retryInterval = 500 * time.Duration(retryTime*retryTime) * time.Millisecond

		logger.Infof("Try Download %v %v time, Retry Interval: %v", t.ID, retryTime, retryInterval)
		t.fileSize, err = GetTransferManager().remoteFileSize(t.url)
		if err != nil {
			logger.Errorf("Try Download %v\n\t[%v/%v] Failed: %v", t.ID, retryTime, downloadRetryTime, err)
			continue
		}

		client, err := GetClient(t.url)
		if err != nil {
			logger.Errorf("Try Download %v\n\t[%v/%v] Failed: %v", t.ID, retryTime, downloadRetryTime, err)
			continue
		}

		if client.SupportRange() {
			err = t.checkLocalFileDupAndDownload(0)
		} else {
			request, _ := client.NewRequest(t.url)
			request.ConnectProgress(t.progress)
			request.ConnectStatusCheck(t.Status)
			err = request.Download(t.localFile)
		}
		if err != nil {
			logger.Errorf("Try Download %v\n\t[%v/%v] Failed: %v", t.ID, retryTime, downloadRetryTime, err)
			continue
		}

		//verfiy MD5
		if 0 != len(t.md5) {
			fileMD5, _ := utils.SysMd5Sum(t.localFile)
			if t.md5 != fileMD5 {
				err = fmt.Errorf("VerifyMD5 %v Failed: remote: %v, check: %v, task.file %v, task.url %v",
					t.ID, fileMD5, t.md5, t.localFile, t.url)
				logger.Errorf("Try Download %v\n\t[%v/%v] Failed: %v", t.ID, retryTime, downloadRetryTime, err)
				os.Remove(t.localFile)
				continue
			}
		}

		t.status = TaskSuccess
		return nil
	}

	if retryTime >= downloadRetryTime {
		logger.Errorf("Try Download %v\n\t[%v/%v] Failed: %v", t.ID, retryTime, downloadRetryTime, err)
		t.status = TaskFailed
	}
	return fmt.Errorf("Download %v Transfer Fialed: %v", t.ID, err)
}

func (t *Transfer) quickDownload() (sucess bool) {
	if 0 == len(t.md5) {
		return false
	}
	fileMD5, _ := utils.SysMd5Sum(t.localFile)
	if t.md5 != fileMD5 {
		return false
	}
	return true
}

func (t *Transfer) checkLocalFileDupAndDownload(duptime int) error {
	logger.Info("Check LocalFile Dup And Download")
	if utils.IsFileExist(t.localFile) {
		if t.quickDownload() {
			logger.Infof("QuickDownload %v success", t.localFile)
			return nil
		}

		t.statusFile = t.localFile + ".tfst"
		if utils.IsFileExist(t.statusFile) {
			return t.breakpointDownloadFile()
		} else {
			if OnDupOverWrite == t.ondup {
				return t.downloadFile()
			} else if OnDupRename == t.ondup {
				t.localFile = t.originLocalFileName + "." + strconv.Itoa(duptime)
				duptime += 1
				//file name dup again, get new
				return t.checkLocalFileDupAndDownload(duptime)
			} else {
				return fmt.Errorf("Error ondup value: %v", t.ondup)
			}
		}
	} else {
		return t.downloadFile()
	}
}

func (t *Transfer) Status() int32 {
	for {
		select {
		case t.status = <-t.taskStatusChan:
			switch t.status {
			case TaskCancel:
				logger.Info("Task", t.ID, " : Cancel")
				return t.status
			case TaskStart:
				logger.Info("Task", t.ID, " : Resume")
			case TaskPause:
				logger.Info("Task", t.ID, " : Pause")
			}

		default:
			runtime.Gosched()
			if t.status != TaskStart {
				break //select
			}
			return t.status //must be TaskStart
		}
	}
	return t.status
}

func (t *Transfer) breakpointDownloadFile() error {
	logger.Infof("Start Breakpoint Download File %v", t.ID)
	tfst, err := LoadTransferStatus(t.statusFile)
	if err != nil {
		logger.Errorf("Breakpoint Download %v failed: %v. ", t.ID, err)
		return err
	}

	logger.Warning(tfst)
	dlfile, err := os.OpenFile(t.localFile, os.O_CREATE|os.O_RDWR, DefaultFileMode)
	defer dlfile.Close()
	if err != nil {
		logger.Errorf("Breakpoint Download %v failed: %v. ", t.ID, err)
		return err
	}

	// TODO:
	//if remote filesize is ZERO, should download yet
	var client Client
	client, err = GetClient(t.url)
	request, _ := client.NewRequest(t.url)
	request.ConnectStatusCheck(t.Status)
	request.ConnectProgress(t.progress)
	for index, slice := range tfst.blockStat {
		logger.Info("BlocakStatus[", index+1, "/", tfst.blockNum, "]: ", slice, "    ", t.ID)
		curblock := int64(index)
		slice.Begin = curblock * tfst.blockSize
		slice.End = (curblock + 1) * tfst.blockSize
		if slice.End > t.fileSize {
			slice.End = t.fileSize
		}
		if slice.Finish < (slice.End - slice.Begin) {
			var data []byte
			retryTime := 0
			for retryTime < rangedownloadRetryTime {
				retryTime++
				data, err = request.DownloadRange(slice.Begin, slice.End)
				if nil != err {
					logger.Errorf("DownloadRange %v failed: %v. Try %v/%v", t.ID, err, retryTime, rangedownloadRetryTime)
					time.Sleep(3000 * time.Duration(retryTime) * time.Millisecond)
				} else {
					break
				}
			}
			if err != nil {
				logger.Errorf("DownloadRange %v failed: %v. Try %v/%v", t.ID, err, retryTime, rangedownloadRetryTime)
				return err
			}
			dlfile.WriteAt(data, int64(slice.Begin))
			dlfile.Sync()
			slice.Finish = slice.End - slice.Begin
			tfst.Sync(curblock, slice)
		}
	}
	tfst.Remove()
	return nil
}

func (t *Transfer) downloadFile() error {
	statusFile := t.localFile + ".tfst"
	blockSize := calcBlockSize(t.fileSize)
	tfst, err := NewTransferStatus(statusFile, blockSize, t.fileSize)
	logger.Warning(tfst)
	if err != nil {
		logger.Error(t.ID, err)
		return err
	} else {
		t.statusFile = statusFile
		return t.breakpointDownloadFile()
	}
}

//calcBlockSize depend remote filesize
//by test, 4M, 8M block has the fast download speed
//and block should not to small
//mini block size 2M
//max block size 8M
func calcBlockSize(remotefilesize int64) int64 {
	blockSize := int64(Mega)

	basicSize := remotefilesize / (8 * Mega)

	switch basicSize {
	case 0:
		blockSize = 2 * Mega
	case 1:
		blockSize = 2 * Mega
	case 2:
		blockSize = 2 * Mega
	case 3:
		blockSize = 2 * Mega
	case 4:
		blockSize = 4 * Mega
	case 5:
		blockSize = 4 * Mega
	case 6:
		blockSize = 4 * Mega
	case 7:
		blockSize = 4 * Mega
	case 8:
		blockSize = 8 * Mega
	default:
		blockSize = 8 * Mega
	}

	return blockSize
}

func (t *Transfer) progress(deta int64, downloaded int64, total int64) {
	t.detaSize += int64(deta)
	t.downloadSize = downloaded
	t.totalSize = t.fileSize
}

func (t *Transfer) String() string {
	return fmt.Sprintf("Taskid: %v\nStatus: %v\nUrl: %v\nMD5: %v\n"+
		"OnDup: %v\nFileSize: %v\nFileName: %v\nLocalFile: %v\nStatusFile: %v\n"+
		"DownLoadSize: %v\nTotalSize: %v", t.ID, t.status, t.url, t.md5,
		t.ondup, t.fileSize, t.fileName, t.localFile, t.statusFile,
		t.downloadSize, t.totalSize)
}
