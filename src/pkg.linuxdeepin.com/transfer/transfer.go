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

package main

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

	TaskPause  = int32(0x14)
	TaskCancel = int32(0x15)
)

const (
	OnDupRename    = int32(0x40)
	OnDupOverWrite = int32(0x41)
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
	t.status = TaskStart
	t.md5 = md5
	t.url = url
	t.originLocalFileName = localFile
	t.localFile = localFile
	t.status = TaskStart
	t.ondup = ondup
	logger.Warningf("[NewTransfer] ID: %v localFile: %v", t.ID, localFile)
	return t, nil
}

func (t *Transfer) Download() error {
	logger.Info("[Download] Start Download url: ", t.url)
	var err error
	retryTime := 3
	for retryTime > 0 {
		retryTime--
		t.fileSize, err = GetService().remoteFileSize(t.url)
		if err != nil {
			logger.Error(err)
			continue
		}

		client, err := GetClient(t.url)
		if err != nil {
			logger.Error(err)
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
			logger.Error(err)
			continue
		}

		//verfiy MD5
		if 0 != len(t.md5) {
			fileMD5, _ := utils.SysMd5Sum(t.localFile)
			if t.md5 != fileMD5 {
				logger.Warningf("[VerifyMD5] dwonload: %v, check: %v, ID %v, task.file %v, task.url %v",
					fileMD5, t.md5, t.ID, t.localFile, t.url)
				os.Remove(t.localFile)
				time.Sleep(100 * time.Millisecond)
				continue
			}
		}

		t.status = TaskSuccess
		return nil
	}

	if retryTime <= 0 {
		t.status = TaskFailed
	}
	return TransferError(fmt.Sprintf("Download Transfer Fialed: %v", t.ID))
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
	logger.Info("[checkLocalFileDupAndDownload] Enter")
	if utils.IsFileExist(t.localFile) {
		if t.quickDownload() {
			logger.Infof("[checkLocalFileDupAndDownload] QuickDownload %v success", t.localFile)
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
				return TransferError(fmt.Sprintf("Error ondup value: %v", t.ondup))
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
				logger.Info("Tasker", t.ID, " : Cancel\n")
				return t.status
			case TaskStart:
				logger.Info("Tasker", t.ID, " : Resume\n")
			case TaskPause:
				logger.Info("Tasker", t.ID, " : Pause\n")
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
	logger.Info("[breakpointDownloadFile] Enter")
	tfst, err := LoadTransferStatus(t.statusFile)
	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Warning(tfst)
	dlfile, err := os.OpenFile(t.localFile, os.O_CREATE|os.O_RDWR, 0755)
	defer dlfile.Close()
	if err != nil {
		logger.Error(err)
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
		logger.Info("BlocakStatus[", index+1, "/", tfst.blockNum, "]: ", slice)
		curblock := int64(index)
		slice.begin = curblock * tfst.blockSize
		slice.end = (curblock + 1) * tfst.blockSize
		if slice.end > t.fileSize {
			slice.end = t.fileSize
		}
		if slice.finish < (slice.end - slice.begin) {
			//TODO, size is not zero
			var data []byte
			data, err = request.DownloadRange(slice.begin, slice.end)
			retryTime := HttpRetryTimes
			err = nil
			for retryTime > 0 {
				retryTime--
				if nil != err {
					time.Sleep(200 * time.Duration(HttpRetryTimes-retryTime) * time.Millisecond)
					data, err = request.DownloadRange(slice.begin, slice.end)
				} else {
					break
				}
			}
			if err != nil {
				logger.Error(err)
				return err
			}
			dlfile.WriteAt(data, int64(slice.begin))
			dlfile.Sync()
			slice.finish = slice.end - slice.begin
			tfst.Sync(curblock, slice)
		}
	}
	//	tfst.Remove()
	return nil
}

func (t *Transfer) downloadFile() error {
	statusFile := t.localFile + ".tfst"
	blockSize := calcBlockSize(t.fileSize)
	tfst, err := NewTransferStatus(statusFile, blockSize, t.fileSize)
	logger.Warning(tfst)
	if err != nil {
		logger.Error(err)
		return err
	} else {
		tfst.Close()
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
