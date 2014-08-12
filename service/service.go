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
	"time"

	"pkg.linuxdeepin.com/lib/dbus"
)

const (
	DBUS_NAME = "com.deepin.download.service"
	DBUS_PATH = "/com/deepin/download/service"
	DBUS_IFC  = "com.deepin.download.service"
)

const (
	C_FREE_PROCESS = int32(0x40)
)

const (
	TS_FINISH = int32(0x10)
)

type Service struct {
	tasks map[string](*Task) //taskid to task

	//control the max gocontinue to download
	maxProcess  int32
	curProcess  int32
	freeProcess chan int32

	//signals

	/*
		@signal Start
			taskid: 任务id
			任务下载开始时发出
	*/
	Start func(taskid string)

	/*
		@signal Update
			taskid: 任务id
		  	process: 下载进度0~100
			speeds 下载速度 Bytes/s
			finish 下载完成的url数目
			total  总共下载的url数目
			downloadSize 已经下载的数据 Byte
			totalSize 总共需要下载的数据 Byte
		每秒钟针对每个任务发出
	*/
	Update func(taskid string, progress int32, speed int32, finish int32, total int32, downloadSize int64, taotalSize int64)

	/*
		@signal Finish
			taskid: 任务id
		任务完成时发出
	*/
	Finish func(taskid string)

	/*
		@signal Pause
			taskid: 任务id
		任务暂停时发出
	*/
	Pause func(taskid string)

	/*
		@signal Stop
			taskid: 任务id
		任务停止时发出, 任务Stop后会被立即删除，无法再获得任务信息，
		一般发出Stop信号，则任务任务失败
	*/
	Stop func(taskid string)

	/*
		@signal Error
			taskid: 任务id
		发生错误时发出
	*/
	Error func(taskid string, errcode int32, errstr string)

	/*
		@signal Resume
			taskid: 任务id
		任务继续时发出
	*/
	Resume func(taskid string)
}

var _service *Service

func GetService() *Service {
	if nil == _service {
		_service = &Service{}
		_service.init()
	}
	return _service
}

func (p *Service) GetDBusInfo() dbus.DBusInfo {
	return dbus.DBusInfo{
		DBUS_NAME,
		DBUS_PATH,
		DBUS_IFC,
	}
}

func (p *Service) startUpdateTaskInfoTimer() {
	//init process update Timer
	logger.Info("[startUpdateTaskInfoTimer] Start Timer")
	timer := time.NewTimer(1 * time.Second)
	for {
		select {
		case <-timer.C:
			p.updateTaskInfo(timer)
		}
	}

}

func (p *Service) updateTaskInfo(timer *time.Timer) {
	//	logger.Info("[updateTaskInfo] Send progress signal per second")
	for taskid, task := range p.tasks {
		progress, curSpeed, finish, total, downloadSize, totalSize := task.RefreshStatus()
		logger.Info(taskid, progress, finish, total, downloadSize, totalSize, curSpeed, "Byte/s")
		p.Update(taskid, int32(progress), int32(curSpeed), int32(finish), int32(total), downloadSize, totalSize)
	}
	timer.Reset(1 * time.Second)
}

func (p *Service) init() {
	logger.Info("[init] Init Service")
	TransferDbus().ConnectFinshReport(p.onTransferFinish)
	TransferDbus().ConnectProcessReport(p.onProcessReport)
	p.curProcess = 0
	p.maxProcess = 8
	p.freeProcess = make(chan int32, p.maxProcess+1)
	p.tasks = map[string](*Task){}
	go p.startUpdateTaskInfoTimer()
}

func (p *Service) onProcessReport(transferID int32, detaSize int64, finishSize int64, totalSize int64) {
	dl := QueryDownloader(transferID)
	if nil == dl {
		logger.Warning("[onProcessReport], nil pkg with transferID: ", transferID)
		return
	}

	for _, task := range dl.refTasks {
		task.UpdateDownloaderStatusHook(dl, detaSize/int64(len(dl.refTasks)), finishSize, totalSize)
	}
}

func (p *Service) onTransferFinish(transferID int32, retCode int32) {
	logger.Infof("[onTransferFinish] Download %v Finist with return Code %v", transferID, retCode)

	dl := QueryDownloader(transferID)
	if nil == dl {
		logger.Warning("[onProcessReport], nil pkg with transferID: ", transferID)
		return
	}

	p.freeProcess <- C_FREE_PROCESS
	p.curProcess = p.curProcess - 1

	for _, task := range dl.refTasks {
		task.FinishDownloaderHook(dl, retCode)
	}
	dl.Finish()
}

//AddTask will add download task to transfer queue and return
//Task is list of debian packages
//pkg is mean single debian package
func (p *Service) AddTask(taskName string, urls []string, sizes []int64, md5s []string, storeDir string) (taskid string) {
	logger.Infof("[AddTask] %v", taskName)
	task := NewTask(taskName, urls, sizes, md5s, storeDir)
	if nil == task {
		return ""
	}
	logger.Infof("[AddTask] %v", taskName)
	task.CB_Finish = p.FinishTask
	task.CB_Cancel = p.CancelTask
	p.tasks[task.ID] = task
	logger.Infof("[AddTask] %v", taskName)
	go p.startTask(task)
	return task.ID
}

//PauseTask will pause Task
func (p *Service) PauseTask(taskid string) {
	p.Pause(taskid)
}

//ResumTask will Resume Task
func (p *Service) ResumeTask(taskid string) {
	p.Resume(taskid)
}

//StopTask will stop Task and DELETE Task
func (p *Service) CancelTask(taskid string) {
	p.removeTask(taskid)
	logger.Infof("[Service] Send task %v Stop signal", taskid)
	p.Stop(taskid)
}

func (p *Service) FinishTask(taskid string) {
	p.removeTask(taskid)
	logger.Infof("[Service] Send task %v Finish signal", taskid)
	p.Finish(taskid)
}

//removeTask will stop Task and DELETE Task
func (p *Service) removeTask(taskid string) {
	logger.Info("[Service] remove task: ", taskid)
	delete(p.tasks, taskid)
}

const (
	E_INIT_TRANSFER_API = int32(0x80)
	E_DOWNLOAD_PKG      = E_INIT_TRANSFER_API + 1
	E_INVAILD_TASKID    = E_INIT_TRANSFER_API + 2
)

func (p *Service) waitProcess() {
	for {
		select {
		case processStatus := <-p.freeProcess:
			switch processStatus {
			case C_FREE_PROCESS:
				return
			}
		}
	}
}

func (p *Service) startTask(task *Task) {
	logger.Infof("[startTask] %v", task)
	task.querySize()

	p.Start(task.ID)
	waitNumber := task.WaitProcessNumber()
	for i := 0; i < waitNumber; i += 1 {
		if p.curProcess >= p.maxProcess {
			p.waitProcess()
		}
		p.curProcess++
		task.StartSingle()
	}
}
