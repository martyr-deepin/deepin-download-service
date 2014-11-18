/*
Copyright (C) 2011~2014 Deepin, Inc.
type FinishReportHandle func(taskid string, statusCode int32)
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
	"sync"
	"time"

	"pkg.linuxdeepin.com/lib/dbus"
)

const (
	ProgressUpdateTime = 900 //MillSecond
)

const (
	TransferManagerDest = "com.deepin.api.Transfer"
	TransferManagerPath = "/com/deepin/api/Transfer"
	TransferManagerIfc  = "com.deepin.api.Transfer"
)

type ProcessReporter func(taskid string, detaBytes int64, finishBytes int64, totalBytes int64)
type FinishReporter func(taskid string, statusCode int32)

type Reporter struct {
	processReportCallBack ProcessReporter
	finishReportCallBack  FinishReporter
}

type TransferManagerLib struct {
	s *TransferManager
}

type TransferManager struct {
	CallBack Reporter           `dbus:"-"`
	Lib      TransferManagerLib `dbus:"-"`

	//Signal
	ProcessReport func(taskid string, detaBytes int64, finishBytes int64, totalBytes int64)
	FinishReport  func(taskid string, statusCode int32)

	MaxTransferNumber int32

	transfers     map[string]*Transfer
	workTransfers *list.List
	waitTransfers *list.List

	worklistLock sync.Mutex
	waitlistLock sync.Mutex
}

var _server *TransferManager

func GetTransferManager() *TransferManager {
	if nil == _server {
		_server = &TransferManager{}
		_server.transfers = map[string]*Transfer{}
		_server.waitTransfers = list.New()
		_server.workTransfers = list.New()
		_server.MaxTransferNumber = 32
		_server.Lib.s = _server
		go _server.startProgressReportTimer()
	}
	return _server
}

const (
	ActionSuccess = int32(0)
	ActionFailed  = int32(1)
)

//transfer is both lib and dbus, it deal with callback and dbus signal

func (r *Reporter) RegisterProcessReporter(f ProcessReporter) {
	r.processReportCallBack = f
}

func (r *Reporter) RegisterFinishReporter(f FinishReporter) {
	r.finishReportCallBack = f
}

func (s *TransferManager) sendProcessReportSignal(taskid string, detaBytes int64, finishBytes int64, totalBytes int64) {
	if nil != s.CallBack.processReportCallBack {
		s.CallBack.processReportCallBack(taskid, detaBytes, finishBytes, totalBytes)
	}
	dbus.Emit(s, "ProcessReport", taskid, detaBytes, finishBytes, totalBytes)
}

func (s *TransferManager) sendFinishReportSignal(taskid string, statusCode int32) {
	if nil != s.CallBack.finishReportCallBack {
		s.CallBack.finishReportCallBack(taskid, statusCode)
	}
	dbus.Emit(s, "FinishReport", taskid, statusCode)
}

func (s *TransferManager) GetDBusInfo() dbus.DBusInfo {
	return dbus.DBusInfo{
		TransferManagerDest,
		TransferManagerPath,
		TransferManagerIfc,
	}
}

func (s *TransferManager) Download(dbusMsg dbus.DMessage, url string, localfile string, md5 string, ondup int32) (retCode int32, taskid string) {
	fmt.Println(dbusMsg, localfile)
	if nil != PermissionVerfiy(dbusMsg.GetSenderPID(), localfile) {
		return ActionFailed, ""
	}

	return s.Lib.Download(url, localfile, md5, ondup)
}

func (sl *TransferManagerLib) Download(url string, localfile string, md5 string, ondup int32) (retCode int32, taskid string) {
	t := sl.s.getTask(url, localfile)
	if nil != t {
		logger.Warningf("Task Exist, Stop Add this Task: %v", localfile)
		return ActionSuccess, t.ID
	}

	logger.Warningf("Add Task:\n\tUrl: %v\n\tFile: %v\n\tMD5: %v\n\tOverwrite: %v", url, localfile, md5, ondup)
	t, _ = NewTransfer(url, localfile, md5, ondup)
	sl.s.transfers[t.ID] = t
	go sl.s.startTask(t)

	return ActionSuccess, t.ID
}

func (s *TransferManager) Resume(taskid string) int32 {
	logger.Info("Resume", taskid)
	task := s.transfers[taskid]
	if task != nil {
		task.taskStatusChan <- TaskStart
		return ActionSuccess
	}
	return ActionFailed
}

func (s *TransferManager) Pause(taskid string) int32 {
	logger.Info("Pause", taskid)
	task := s.transfers[taskid]
	if task != nil {
		task.taskStatusChan <- TaskPause
		return ActionSuccess
	}
	return ActionFailed
}

func (s *TransferManager) Cancel(taskid string) int32 {
	logger.Warning("Cancel", taskid)
	task := s.transfers[taskid]
	delete(s.transfers, taskid)
	if task != nil {
		s.sendFinishReportSignal(taskid, TaskFailed)
		go func() { task.taskStatusChan <- TaskCancel }()
		return ActionSuccess
	}
	return ActionFailed
}

func (s *TransferManager) Close() {
	quitAllFtpClient()
}

func (s *TransferManager) QuerySize(url string) int64 {
	size, err := s.remoteFileSize(url)
	if nil != err {
		return 0
	}
	return size
}

//task manager

func (s *TransferManager) TransferCount() int32 {
	return int32(len(s.transfers))
}

func (s *TransferManager) ListTransfer() []string {
	var transferlist []string
	for _, v := range s.transfers {
		transferlist = append(transferlist, v.ID)
	}
	return transferlist
}

//DumpTransfer is for debug
func (s *TransferManager) GetTransfer(taskid string) string {
	return fmt.Sprintf("%v", s.transfers[taskid])
}

func (s *TransferManager) getTask(url string, localfile string) *Transfer {
	for _, task := range s.transfers {
		if (task.url == url) && (task.localFile == localfile) {
			return task
		}
	}
	return nil
}

func (s *TransferManager) remoteFileSize(url string) (int64, error) {
	client, err := GetClient(url)

	if err != nil {
		logger.Error("Get Client Failed: ", err)
		return 0, err
	}

	size, err := client.QuerySize(url)
	if err != nil {
		logger.Error("QuerySize Failed: ", err)
		return 0, err
	}

	return size, nil
}

func (s *TransferManager) startTask(t *Transfer) {
	s.worklistLock.Lock()
	defer s.worklistLock.Unlock()
	s.waitlistLock.Lock()
	defer s.waitlistLock.Unlock()
	if int32(s.workTransfers.Len()) < s.MaxTransferNumber {
		t.element = s.workTransfers.PushBack(t)
		go s.download(t)
	} else {
		t.element = s.waitTransfers.PushBack(t)
	}
}

func (s *TransferManager) finishTask(t *Transfer) {
	s.worklistLock.Lock()
	defer s.worklistLock.Unlock()
	s.waitlistLock.Lock()
	defer s.waitlistLock.Unlock()

	logger.Warningf("FinishTask:\n\tID: %v\n\tUrl: %v\n\tStatus: %v", t.ID, t.url, t.status)
	s.sendFinishReportSignal(t.ID, t.status)

	if nil != t.element {
		s.workTransfers.Remove(t.element)
	}
	delete(s.transfers, t.ID)

	// Start a new task
	element := s.waitTransfers.Front()
	if nil == element {
		return
	}

	value := s.waitTransfers.Remove(element)
	if nt, ok := value.(*Transfer); ok {
		go s.startTask(nt)
	}
}

func (s *TransferManager) download(t *Transfer) {
	logger.Infof("Download %v", t.ID)
	defer s.finishTask(t)

	err := t.Download()
	if err != nil {
		logger.Error(t.ID, err)
		t.status = TaskFailed
	} else {
		t.status = TaskSuccess
	}
}

func (s *TransferManager) startProgressReportTimer() {
	timer := time.NewTimer(ProgressUpdateTime * time.Millisecond)
	for {
		select {
		case <-timer.C:
			s.handleProgressReport()
			timer.Reset(ProgressUpdateTime * time.Millisecond)
		}
	}
}

func (s *TransferManager) handleProgressReport() {
	for element := s.workTransfers.Front(); element != nil; element = element.Next() {
		if t, ok := element.Value.(*Transfer); ok {
			//logger.Warning("Report Progress of", task.ID, " size ", task.detaSize)
			s.sendProcessReportSignal(t.ID, t.detaSize, t.downloadSize, t.fileSize)
			t.detaSize = 0
		}
	}
}
