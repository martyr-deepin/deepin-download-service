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
	"errors"
	"time"

	"pkg.linuxdeepin.com/lib/dbus"
)

const (
	ProgressUpdateTime = 900 //MillSecond
)

type ProcessReportHandle func(taskid string, detaBytes int64, finishBytes int64, totalBytes int64)
type FinishReportHandle func(taskid string, statusCode int32)

type Service struct {
	//Signal
	ProcessReport func(taskid string, detaBytes int64, finishBytes int64, totalBytes int64)
	FinishReport  func(taskid string, statusCode int32)

	cbProcessReport func(taskid string, detaBytes int64, finishBytes int64, totalBytes int64)
	cbFinishReport  func(taskid string, statusCode int32)

	MaxTransferNumber int32

	transfers     map[string]*Transfer
	workTransfers *list.List
	waitTransfers *list.List
}

const (
	ServiceDest = "com.deepin.api.Transfer"
	ServicePath = "/com/deepin/api/Transfer"
	ServiceIfc  = "com.deepin.api.Transfer"
)

func (s *Service) GetDBusInfo() dbus.DBusInfo {
	return dbus.DBusInfo{
		ServiceDest,
		ServicePath,
		ServiceIfc,
	}
}

var _server *Service

func GetService() *Service {
	if nil == _server {
		_server = &Service{}
		_server.transfers = map[string]*Transfer{}
		_server.waitTransfers = list.New()
		_server.workTransfers = list.New()
		_server.MaxTransferNumber = 32
		go _server.startProgressReportTimer()
	}
	return _server
}

func TransferError(msg string) (terr error) {
	return errors.New("[ServiceError]: " + msg)
}

const (
	ActionSuccess = int32(0)
	ActionFailed  = int32(1)
)

//transfer is both lib and dbus, it deal with callback and dbus signal
func (s *Service) RegisterProcessReporter(f ProcessReportHandle) {
	s.cbProcessReport = f
}

func (s *Service) RegisterFinishReporter(f FinishReportHandle) {
	s.cbFinishReport = f
}

func (s *Service) sendProcessReportSignal(taskid string, detaBytes int64, finishBytes int64, totalBytes int64) {
	if nil != s.cbProcessReport {
		s.cbProcessReport(taskid, detaBytes, finishBytes, totalBytes)
	}
	dbus.Emit(s, "ProcessReport", taskid, detaBytes, finishBytes, totalBytes)
}

func (s *Service) sendFinishReportSignal(taskid string, statusCode int32) {
	if nil != s.cbFinishReport {
		s.cbFinishReport(taskid, statusCode)
	}
	dbus.Emit(s, "FinishReport", taskid, statusCode)

}

func (s *Service) Resume(taskid string) int32 {
	logger.Info("[Resume]", taskid)
	task := s.transfers[taskid]
	if task != nil {
		task.taskStatusChan <- TaskStart
		return ActionSuccess
	}
	return ActionFailed
}

func (s *Service) Pause(taskid string) int32 {
	logger.Info("[Pause]", taskid)
	task := s.transfers[taskid]
	if task != nil {
		task.taskStatusChan <- TaskPause
		return ActionSuccess
	}
	return ActionFailed
}

func (s *Service) Cancel(taskid string) int32 {
	logger.Warning("[Cancel]", taskid)
	task := s.transfers[taskid]
	delete(s.transfers, taskid)
	if task != nil {
		s.sendFinishReportSignal(taskid, TaskFailed)
		go func() { task.taskStatusChan <- TaskCancel }()
		return ActionSuccess
	}
	return ActionFailed
}

/*
url: url to download
localfile: path for download file in local disk
ondup: 0 overwrite when dup
        1 make a new name

return Download Status
*/
func (s *Service) Download(url string, localFile string, md5 string, ondup int32) (retCode int32, taskid string) {
	t := s.getTask(url, localFile)
	if nil != t {
		logger.Warningf("[Download] Task Exist, Stop Add this Task: %v", localFile)
		return ActionSuccess, t.ID
	}

	logger.Warning("[Download] Task ADD")
	t, _ = NewTransfer(url, localFile, md5, ondup)
	s.transfers[t.ID] = t
	go s.startTask(t)

	return ActionSuccess, t.ID
}

func (s *Service) QuerySize(url string) int64 {
	size, err := s.remoteFileSize(url)
	if nil != err {
		return 0
	}
	return size
}

//task manager

func (s *Service) TotalTaskCount() int64 {
	return int64(len(s.transfers))
}

func (s *Service) Exit() {
	QuitAllFtpClient()
}

func (s *Service) getTask(url string, localfile string) *Transfer {
	for _, task := range s.transfers {
		if (task.url == url) && (task.localFile == localfile) {
			return task
		}
	}
	return nil
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
func (s *Service) remoteFileSize(url string) (int64, error) {
	client, err := GetClient(url)

	if err != nil {
		logger.Error("[remotefilesize]Get Remove Files Failed: ", err)
		return 0, err
	}

	size, err := client.QuerySize(url)
	if err != nil {
		logger.Error("[remotefilesize]Get Remove Files Failed: ", err)
		return 0, err
	}

	return size, nil
}

func (s *Service) startTask(t *Transfer) {
	if int32(s.workTransfers.Len()) < s.MaxTransferNumber {
		t.element = s.workTransfers.PushBack(t)
		go s.download(t)
	} else {
		t.element = s.waitTransfers.PushBack(t)
	}
}

func (s *Service) finishTask(t *Transfer) {
	logger.Warningf("[finishTask]: %v %v %v", t.ID, t.url, t.status)
	s.sendFinishReportSignal(t.ID, t.status)

	if nil != t.element {
		s.workTransfers.Remove(t.element)
	}
	delete(s.transfers, t.ID)

	// TODO: exit transfer if all.transfers finish

	// Start a new task
	element := s.waitTransfers.Front()

	if nil == element {
		return
	}
	value := s.waitTransfers.Remove(element)
	if nt, ok := value.(*Transfer); ok {
		s.startTask(nt)
	}
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
func (s *Service) download(t *Transfer) {
	logger.Info("[download] Start Download url: ", t.url)
	var err error
	defer s.finishTask(t)

	err = t.Download()
	if err != nil {
		logger.Error(err)
		t.status = TaskFailed
	} else {
		t.status = TaskSuccess
	}
}

func (s *Service) startProgressReportTimer() {
	timer := time.NewTimer(ProgressUpdateTime * time.Millisecond)
	for {
		select {
		case <-timer.C:
			s.handleProgressReport()
			timer.Reset(ProgressUpdateTime * time.Millisecond)
		}
	}
}

func (s *Service) handleProgressReport() {
	//	logger.Warningf("workTask Len: %v", t.workTransfers.Len())
	for element := s.workTransfers.Front(); element != nil; element = element.Next() {
		if t, ok := element.Value.(*Transfer); ok {
			//logger.Warning("Report Progress of", task.ID, " size ", task.detaSize)
			s.sendProcessReportSignal(t.ID, t.detaSize, t.downloadSize, t.fileSize)
			t.detaSize = 0
		}
	}
}
