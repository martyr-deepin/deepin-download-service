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
	tasks     map[string](*Task) //taskid to task
	workTasks map[string](*Task)
	//control the max gocontinue to download
	maxProcess    int32
	maxTask       int32
	taskQueue     chan *Task
	downloadQueue chan *Downloader
	//signals

	/*
		@signal Wait
			taskid: 任务id
			任务下载未开始时发出
	*/
	Wait func(taskid string)

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
	TransferDbus().ConnectFinishReport(p.onTransferFinish)
	TransferDbus().ConnectProcessReport(p.onProcessReport)
	p.maxProcess = 6
	p.maxTask = 1
	p.tasks = map[string](*Task){}
	p.workTasks = map[string](*Task){}

	p.taskQueue = make(chan *Task, p.maxTask)
	p.downloadQueue = make(chan *Downloader, p.maxProcess)
	go p.taskDownlowner()
	go p.startUpdateTaskInfoTimer()
	go p.downloaderDispatch()
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

	//when the finish task is full, this chan will block
	//but when you start a new task, it will recover
	//let it write async
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
		logger.Error("[AddTask] %v Failed", taskName)
		return ""
	}
	task.CB_Finish = p.finishTask
	task.CB_Cancel = p.cancelTask
	p.tasks[task.ID] = task
	go p.startTask(task)
	return task.ID
}

//PauseTask will pause Task
func (p *Service) PauseTask(taskid string) {
	task := p.tasks[taskid]

	if nil == task {
		logger.Warning("[PasueTask] nil task with taskid: ", taskid)
		return
	}

	logger.Infof("[PauseTask] %v", taskid)
	task.Pause()

	p.Pause(taskid)
}

//ResumTask will Resume Task
func (p *Service) ResumeTask(taskid string) {
	task := p.tasks[taskid]

	if nil == task {
		logger.Warning("[ResumeTask] nil task with taskid: ", taskid)
		return
	}
	logger.Infof("[ResumeTask] %v", taskid)
	task.Resume()
	p.Resume(taskid)
}

//StopTask will stop Task and DELETE Task
func (p *Service) StopTask(taskid string) {
	task := p.tasks[taskid]
	if nil == task {
		logger.Warning("[finishTask] nil task with taskid %v", taskid)
		return
	}
	task.Stop()
	p.removeTask(taskid)
	logger.Infof("[Service] Send task %v Stop signal", taskid)
	p.Stop(taskid)
}

//StopTask will stop Task and DELETE Task
func (p *Service) cancelTask(taskid string, errCode int32, errStr string) {
	p.removeTask(taskid)
	logger.Infof("[Service] Send task %v Stop signal", taskid)
	p.Error(taskid, errCode, errStr)
	p.Stop(taskid)
}

func (p *Service) finishTask(taskid string) {
	p.removeTask(taskid)
	logger.Infof("[Service] Send task %v Finish signal", taskid)
	p.Finish(taskid)
}

//removeTask will stop Task and DELETE Task
func (p *Service) removeTask(taskid string) {
	logger.Info("[Service] remove task: ", taskid)
	delete(p.tasks, taskid)
	delete(p.workTasks, taskid)
}

const (
	E_INIT_TRANSFER_API = int32(0x80)
	E_DOWNLOAD_PKG      = E_INIT_TRANSFER_API + 1
	E_INVAILD_TASKID    = E_INIT_TRANSFER_API + 2
)

func (p *Service) startTask(task *Task) {
	logger.Infof("[startTask] %v", task)
	p.Wait(task.ID)
	task.querySize()

	p.taskQueue <- task
}

func (p *Service) taskDownlowner() {
	for {
		select {
		case task := <-p.taskQueue:
			//control the task
			if (nil != task) && task.Vaild() {
				for len(p.workTasks) >= int(p.maxTask) {
					time.Sleep(1 * time.Second)
				}
				p.workTasks[task.ID] = task
				logger.Warning("Start Single of task ", task.ID)
				waitNumber := task.WaitProcessNumber()
				logger.Warning("waitNumber", waitNumber)
				sendTaskStart := false
				for i := 0; i < waitNumber; i += 1 {
					dl := task.StartSingle()
					if (nil != dl) && (DownloaderWait == dl.status) {
						logger.Warning("send start", dl.ID)
						p.downloadQueue <- dl
						logger.Warning("send end", dl.ID)
					}
					if !sendTaskStart {
						logger.Warning("send task start", task.ID)
						p.Start(task.ID)
						sendTaskStart = true
					}
				}
			} else {
				logger.Warning("Exit invaild task ", task.ID)
			}

		}
	}
}

func (p *Service) downloaderDispatch() {
	for {
		select {
		case dl := <-p.downloadQueue:
			logger.Warning("start dl", dl.ID)
			dl.Start()
		}
	}
}
