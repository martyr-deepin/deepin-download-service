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
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"pkg.linuxdeepin.com/lib/dbus"
)

type Transfer struct {
	//Signal
	ProcessReport func(taskid int32, beginBytes int64, endBytes int64)
	FinshReport   func(taskid int32, statusCode int32)

	//to get an unique taskid
	taskidgen int32
	tasks     map[int32]*TranferTaskInfo
}

const (
	TRANSFER_DEST = "com.deepin.api.Transfer"
	TRANSFER_PATH = "/com/deepin/api/Transfer"
	TRANSFER_IFC  = "com.deepin.api.Transfer"
)

func (t *Transfer) GetDBusInfo() dbus.DBusInfo {
	return dbus.DBusInfo{
		TRANSFER_DEST,
		TRANSFER_PATH,
		TRANSFER_IFC,
	}
}

var _transfer *Transfer

func GetTransfer() *Transfer {
	if nil == _transfer {
		_transfer = &Transfer{}
		_transfer.taskidgen = 0
		_transfer.tasks = map[int32]*TranferTaskInfo{}
	}
	return _transfer
}

func TransferError(msg string) (terr error) {
	return errors.New("TransferError: " + msg)
}

const (
	TASK_SUCCESS = int32(0)
	TASK_FAILED  = int32(1)
)

const (
	ACTION_SUCCESS = int32(0)
	ACTION_FAILED  = int32(1)
)

const (
	TASK_ST_RUNING = int32(0)
	TASK_ST_PAUSE  = int32(1)
	TASK_ST_CANCEL = int32(2)
)

func (t *Transfer) Resume(taskid int32) int32 {
	taskinfo := t.tasks[taskid]
	logger.Info("Resume", taskid)
	if taskinfo != nil {
		taskinfo.taskchan <- TASK_ST_RUNING
		logger.Info("Resume", taskid)
		return ACTION_SUCCESS
	}
	return ACTION_FAILED
}

func (t *Transfer) Pause(taskid int32) int32 {
	taskinfo := t.tasks[taskid]
	logger.Info("Pause", taskid)
	if taskinfo != nil {
		taskinfo.taskchan <- TASK_ST_PAUSE
		return ACTION_SUCCESS
	}
	return ACTION_FAILED
}

func (t *Transfer) Cancel(taskid int32) int32 {
	taskinfo := t.tasks[taskid]
	if taskinfo != nil {
		taskinfo.taskchan <- TASK_ST_CANCEL
		return ACTION_SUCCESS
	}
	return ACTION_FAILED
}

/*
url: url to download
localfile: path for download file in local disk
ondup: 0 overwrite when dup
        1 make a new name

return Download Status
*/
func (t *Transfer) Download(url string, localfile string, ondup int32) (taskid int32) {
	taskinfo := &TranferTaskInfo{}
	taskinfo.taskid = t.newTaskid()
	taskinfo.taskchan = make(chan int32, 20)
	taskinfo.url = url
	taskinfo.originLocalFilename = localfile
	taskinfo.localFile = localfile
	taskinfo.status = TASK_ST_RUNING
	t.tasks[taskinfo.taskid] = taskinfo

	go t.download(taskinfo, ondup)

	return taskinfo.taskid
}

func (t *Transfer) QuerySize(url string) int64 {
	size, err := t.remoteFileSize(url)
	if nil != err {
		return 0
	}
	return size
}

const (
	SIZE_1K = 1024
	SIZE_1M = SIZE_1K * SIZE_1K
)

type TranferTaskInfo struct {
	taskid              int32
	url                 string
	fileSize            int64
	fileName            string
	originLocalFilename string
	localFile           string
	dlStatusFile        string

	status   int32
	taskchan chan int32
}

type DownloadStatusInfo struct {
	fileSize  int64
	blockSize int64
	blockNum  int64
	blockStat []byte
}

/*
@description
    generate a new tranfer taskid
@input

@return
    a unique taskid in all transfer task
*/
func (t *Transfer) newTaskid() int32 {
	t.taskidgen += 1
	return t.taskidgen
}

/*
@description
    check if the file exist
@input
    filename: the full path of file
@return
    true if file exist, otherwise false
*/
func isFileExist(filename string) bool {
	isExist := bool(false)
	file, err := os.Open(filename)
	if err != nil {
		isExist = false
	} else {
		isExist = true
	}
	file.Close()
	return isExist
}

/*
@description
    check if the file exist
@input
    url: the url of remote file
@return
    0 if remote server do not support Content-Length Header or other errors
    otherwise return the remote file size
*/
func (t *Transfer) remoteFileSize(url string) (int64, error) {
	logger.Info("remoteFileSize enter")
	client := &http.Client{}
	reqest, _ := http.NewRequest("GET", url, nil)
	fileSize := int64(0)
	response, _ := client.Do(reqest)
	if nil == response {
		return fileSize, TransferError("Http Request Error, Url: " + url)
	}

	if response.StatusCode == 200 {
		fileSizeStr := string(response.Header.Get("Content-Length"))
		logger.Warning(fileSizeStr)
		size, err := strconv.Atoi(fileSizeStr)
		if err != nil {
			logger.Error("Set file Size")
			fileSize = 0
		}
		fileSize = int64(size)
		if 0 == fileSize {
			logger.Warning("Maybe Server Do not support Content-Length")
		}
		return int64(fileSize), nil
	}
	return int64(fileSize), TransferError("Get http file error. status code: " + strconv.Itoa(response.StatusCode))
}

/*
@description
    check if the file exist
@input
    url: the url of remote file
@return
    0 if remote server do not support Content-Length Header
    otherwise return the remote file size
*/
func (t *Transfer) download(taskinfo *TranferTaskInfo, ondup int32) error {
	logger.Info("Start Download url: ", taskinfo.url)
	err := errors.New("")
	taskinfo.fileSize, err = t.remoteFileSize(taskinfo.url)
	if err != nil {
		logger.Error(err)
		return err
	}

	err = t.checkLocalFileDupAndDownload(taskinfo, ondup, 0)
	if err != nil {
		logger.Error("Report download task failed", err)
		t.FinshReport(taskinfo.taskid, taskinfo.status)
		return err
	}
	//Only Success Remove
	//removedlstFile(taskinfo.dlStatusFile)
	logger.Warningf("FinishReport: %v", taskinfo)
	t.FinshReport(taskinfo.taskid, taskinfo.status)
	return nil
}

/*
@description
    check localfile, if exist, append a num to the end of the localfile name or
    overwrite depend ondup
@input
    taskinfo: the info about download task
    ondup:    0 : overwrite; 1 : crete new localfile name
    duptime:  if the the new filename dup again, it increas, to generate a new filename
@return
    errors when download file
*/
func (t *Transfer) checkLocalFileDupAndDownload(taskinfo *TranferTaskInfo, ondup int32, duptime int) error {
	logger.Info("checkDupDownload enter")
	if isFileExist(taskinfo.localFile) {
		taskinfo.dlStatusFile = taskinfo.localFile + ".dlst"
		if isFileExist(taskinfo.dlStatusFile) {
			return t.breakpointDownloadFile(taskinfo)
		} else {
			if 0 == ondup {
				return t.downloadFile(taskinfo)
			} else if 1 == ondup {
				taskinfo.localFile = taskinfo.originLocalFilename + "." + strconv.Itoa(duptime)
				duptime += 1
				//file name dup again, get new
				return t.checkLocalFileDupAndDownload(taskinfo, ondup, duptime)
			} else {
				return TransferError("Error ondup value: ")
			}
		}
	} else {
		return t.downloadFile(taskinfo)
	}
}

/*
@description
    create new download status file *.dlst
    file fomat{
        [FileSize int32 ][block Size int32 ][block Num int32 ][All Block []Byte]
    }
@input
    dlStatusFile: download status file name
    blockSize:    block size every time download
    fileSize:     total file size in server
@return
    errors
*/
func (t *Transfer) newdlstFile(dlStatusFile string, blockSize int64, fileSize int64) error {
	logger.Info("newdlstFile params: dlstatusFile, blockSize, fileSize", dlStatusFile, blockSize, fileSize)
	f, err := os.Create(dlStatusFile)
	defer f.Close()
	if err != nil {
		logger.Error(err)
		return err
	} else {
		blockNum := fileSize / blockSize
		if blockNum*blockSize < fileSize {
			blockNum += 1
		}
		dlst := DownloadStatusInfo{
			fileSize,
			blockSize,
			blockNum,
			make([]byte, blockNum),
		}
		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.LittleEndian, fileSize)
		err = binary.Write(buf, binary.LittleEndian, dlst.blockSize)
		err = binary.Write(buf, binary.LittleEndian, dlst.blockNum)
		err = binary.Write(buf, binary.LittleEndian, dlst.blockStat)
		_, err = f.Write(buf.Bytes())
		if err != nil {
			logger.Error("binary.Write failed:", err)
			return err
		}
	}
	return nil
}

func removedlstFile(dlStatusFile string) error {
	logger.Info("removedlstFile enter")
	err := os.Remove(dlStatusFile)
	return err
}

/*
@description
    create new download status file *.dlst
    file fomat{
        [FileSize int32 ][block Size int32 ][block Num int32 ][All Block []Byte]
    }
@input
    dlStatusFile: download status file name
    dlstInfo:     download status info
@return
    errors
*/
func savedlstFile(dlStatusFile string, dlstInfo DownloadStatusInfo) error {
	logger.Info("savedlstFile enter")
	f, err := os.OpenFile(dlStatusFile, os.O_TRUNC|os.O_RDWR, 0666)
	defer f.Close()
	if err != nil {
		logger.Error(err)
		return err
	} else {
		buf := new(bytes.Buffer)
		err = binary.Write(buf, binary.LittleEndian, dlstInfo.fileSize)
		err = binary.Write(buf, binary.LittleEndian, dlstInfo.blockSize)
		err = binary.Write(buf, binary.LittleEndian, dlstInfo.blockNum)
		err = binary.Write(buf, binary.LittleEndian, dlstInfo.blockStat)
		_, err = f.Write(buf.Bytes())
		if err != nil {
			logger.Error("binary.Write failed:", err)
			return err
		}
		f.Sync()
	}
	return nil

}

/*
@description
    load download status file *.dlst
    file fomat{
        [FileSize int32 ][block Size int32 ][block Num int32 ][All Block []Byte]
    }
@input
    dlStatusFile: download status file name
@return
    DownloadStatusInfo: download status info read from file
    error: errors
*/
func loaddlstFile(dlStatusFile string) (DownloadStatusInfo, error) {
	logger.Info("loaddlstFile enter")
	dlstfile, err := os.Open(dlStatusFile)
	defer dlstfile.Close()
	dlst := new(DownloadStatusInfo)
	if err != nil {
		logger.Error(err)
		return *dlst, err
	} else {
		stats, err := dlstfile.Stat()
		if err != nil {
			logger.Error(err)
			return *dlst, err
		}

		size := stats.Size()
		bytesbuf := make([]byte, size)
		bufr := bufio.NewReader(dlstfile)
		_, err = bufr.Read(bytesbuf)
		if err != nil {
			logger.Error(err)
			return *dlst, err
		}

		buf := bytes.NewReader(bytesbuf)
		err = binary.Read(buf, binary.LittleEndian, &dlst.fileSize)
		err = binary.Read(buf, binary.LittleEndian, &dlst.blockSize)
		err = binary.Read(buf, binary.LittleEndian, &dlst.blockNum)
		dlst.blockStat = make([]byte, dlst.blockNum)
		err = binary.Read(buf, binary.LittleEndian, &dlst.blockStat)
		if err != nil {
			logger.Error(err)
			return *dlst, err
		}
	}
	return *dlst, nil
}

func checkTaskStatus(taskinfo *TranferTaskInfo) int32 {
	for {
		select {
		case taskinfo.status = <-taskinfo.taskchan:
			switch taskinfo.status {
			case TASK_ST_CANCEL:
				logger.Info("Tasker", taskinfo.taskid, " : Cancel\n")
				return taskinfo.status
			case TASK_ST_RUNING:
				logger.Info("Tasker", taskinfo.taskid, " : Resume\n")
			case TASK_ST_PAUSE:
				logger.Info("Tasker", taskinfo.taskid, " : Pause\n")
			}

		default:
			runtime.Gosched()
			if taskinfo.status != TASK_ST_RUNING {
				break //select
			}
			return taskinfo.status //must be TASK_ST_RUNING
		}
	}
	return taskinfo.status
}

/*
@description
    breakpointDownloadFile
@input
    taskinfo: download task info
@return
    error: errors
*/
func (t *Transfer) breakpointDownloadFile(taskinfo *TranferTaskInfo) error {
	logger.Info("breakpointDownloadFile enter")
	dlst, err := loaddlstFile(taskinfo.dlStatusFile)
	if err != nil {
		logger.Error(err)
		return err
	}
	dlfile, err := os.OpenFile(taskinfo.localFile, os.O_CREATE|os.O_RDWR, 0755)
	defer dlfile.Close()
	if err != nil {
		logger.Error(err)
		return err
	}

	// TODO:
	//if remote filesize is ZERO, should download yet

	for index, value := range dlst.blockStat {
		logger.Info("BlocakStatus[", index+1, "/", dlst.blockNum, "]: ", value)
		if 0 == value {
			curblock := int64(index)
			beginByte := curblock * dlst.blockSize
			endByte := (curblock + 1) * dlst.blockSize
			//TODO, size is not zero
			if endByte > taskinfo.fileSize {
				endByte = taskinfo.fileSize
			}

			//			logger.Info("DownloadRange: bytes: ", beginByte, "-", endByte, "/", taskinfo.fileSize)

			data, err := downloadRange(taskinfo, beginByte, endByte-1)
			if err != nil {
				logger.Error(err)
				return err
			}
			dlfile.WriteAt(data, int64(beginByte))
			dlfile.Sync()
			t.ProcessReport(taskinfo.taskid, beginByte, endByte)
			dlst.blockStat[index] = 1
			savedlstFile(taskinfo.dlStatusFile, dlst)
		}
	}
	//make sure success download when return here
	removedlstFile(taskinfo.dlStatusFile)
	return nil
}

/*
@description
    just download File with create new localfile and dlst file
@input
    taskinfo: download task info
@return
    error: errors
*/
func (t *Transfer) downloadFile(taskinfo *TranferTaskInfo) error {
	dlStatusFile := taskinfo.localFile + ".dlst"
	blockSize := calcBlockSize(taskinfo.fileSize)
	err := t.newdlstFile(dlStatusFile, blockSize, taskinfo.fileSize)
	if err != nil {
		logger.Error(err)
		return err
	} else {
		taskinfo.dlStatusFile = dlStatusFile
		return t.breakpointDownloadFile(taskinfo)
	}
}

/*
@description
    calc download BlockSize depend remote filesize
    by test, 4M, 8M block has the fast download speed
    and block should not to small
    mini block size 2M
    max block size 8M
@input
    remotefilesize: remote file size
@return
    error: errors
*/

func calcBlockSize(remotefilesize int64) int64 {
	blockSize := int64(SIZE_1M)

	basicSize := remotefilesize / (8 * SIZE_1M)

	switch basicSize {
	case 0:
		blockSize = 2 * SIZE_1M
	case 1:
		blockSize = 2 * SIZE_1M
	case 2:
		blockSize = 2 * SIZE_1M
	case 3:
		blockSize = 2 * SIZE_1M
	case 4:
		blockSize = 4 * SIZE_1M
	case 5:
		blockSize = 4 * SIZE_1M
	case 6:
		blockSize = 4 * SIZE_1M
	case 7:
		blockSize = 4 * SIZE_1M
	case 8:
		blockSize = 8 * SIZE_1M
	default:
		blockSize = 8 * SIZE_1M
	}

	return blockSize
}

/*
@description
    download a file with rangeBegin to rangeEnd
@input
    url: url for download file
    rangeBegin: pos of start bytes
    rangeEnd: pos of end bytes
@return
    error: errors
*/
func downloadRange(taskinfo *TranferTaskInfo, rangeBegin int64, rangeEnd int64) ([]byte, error) {
	client := &http.Client{}
	url := taskinfo.url
	reqest, _ := http.NewRequest("GET", url, nil)
	bytestr := "bytes=" + strconv.Itoa(int(rangeBegin)) + "-" + strconv.Itoa(int(rangeEnd))
	reqest.Header.Set("Range", bytestr)
	response, _ := client.Do(reqest)

	if (response.StatusCode == 200) || (response.StatusCode == 206) {
		//		logger.Info("Response Headers: ", response.Header)
		//		logger.Info("Read Content-Range:", response.Header.Get("Content-Range"))
		capacity := rangeEnd - rangeBegin
		buf := make([]byte, 0, capacity*2)
		for {
			//checkTaskStatus will block if status is pause
			if TASK_ST_CANCEL == checkTaskStatus(taskinfo) {
				return buf, TransferError("Download Cancel")
			}
			m, e := response.Body.Read(buf[len(buf):cap(buf)])
			buf = buf[0 : len(buf)+m]
			if e == io.EOF {
				logger.Info("Read io.EOF: ", len(buf))
				break
			}
			if e != nil {
				time.Sleep(4 * time.Millisecond)
				logger.Info("Read e: ", e)
				return buf, e
			}
		}

		//contents, err := ioutil.ReadAll(response.Body)
		//logger.Info("Read Content-Range end")
		return buf, nil
	}
	return []byte(""), TransferError("Download url error, Http statuscode: " + strconv.Itoa(response.StatusCode))
}
