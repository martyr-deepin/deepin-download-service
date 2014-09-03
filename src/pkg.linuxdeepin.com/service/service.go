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
	"bytes"
	"os/exec"
	"strings"
	"time"

	"pkg.linuxdeepin.com/lib/dbus"
)

const (
	DBUS_NAME = "com.deepin.download.service"
	DBUS_PATH = "/com/deepin/download/service"
	DBUS_IFC  = "com.deepin.download.service"
)

const (
	TASK_START    = int32(0x10)
	TASK_SUCCESS  = int32(0x11)
	TASK_FAILED   = int32(0x12)
	TASK_NOT_EXIT = int32(0x13)
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

func (p *Service) startCheckTaskInfoTimer() {
	//init process update Timer
	logger.Info("[startCheckTaskInfoTimer] Start Timer")
	timer := time.NewTimer(20 * time.Second)
	for {
		select {
		case <-timer.C:
			p.checkTaskInfo()
			timer.Reset(20 * time.Second)
		}
	}
}

func VerifyMD5(file string) string {
	cmdline := "md5sum -b " + file
	cmd := exec.Command("/bin/sh", "-c", cmdline)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if nil != err {
		logger.Warning("[VerifyMD5] Error: ", err)
	}
	logger.Warning("[VerifyMD5] ", out.String())
	md5 := strings.Split(out.String(), " ")[0]
	return md5
}

func (p *Service) checkTaskInfo() {
	logger.Warning("checkTaskInfo")
	for _, task := range p.workTasks {
		for _, dl := range task.downloaders {
			if 0 != len(dl.transferID) {
				status, err := TransferDbus().QueryTaskStatus(dl.transferID)
				if nil != err {
					logger.Warning("checkTaskInfo error ", err)
					status = TASK_NOT_EXIT
				}
				switch status {
				case TASK_START:
					logger.Warning("task start")
					continue
				case TASK_SUCCESS:
					logger.Warning("task success")
					p.onTransferFinish(dl.transferID, TASK_SUCCESS)
				case TASK_FAILED:
					logger.Warning("task failed")
					p.onTransferFinish(dl.transferID, TASK_FAILED)
				case TASK_NOT_EXIT:
					logger.Warning("task not exit", dl.transferID)
					if dl.md5 == VerifyMD5(dl.storeDir+"/"+dl.fileName) {
						logger.Warning("task not exit", dl.transferID, dl.md5)
						p.onTransferFinish(dl.transferID, TASK_SUCCESS)
					} else {
						p.onTransferFinish(dl.transferID, TASK_FAILED)
						logger.Warning("task error")
					}
				}
			}
		}
	}
}

func (p *Service) updateTaskInfo(timer *time.Timer) {
	//	logger.Info("[updateTaskInfo] Send progress signal per second")
	for taskid, task := range p.tasks {
		progress, curSpeed, finish, total, downloadSize, totalSize := task.RefreshStatus()
		//		logger.Info(taskid, progress, finish, total, downloadSize, totalSize, curSpeed, "Byte/s")
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
	go p.startCheckTaskInfoTimer()
}

func (p *Service) onProcessReport(transferID string, detaSize int64, finishSize int64, totalSize int64) {
	dl := QueryDownloader(transferID)
	if nil == dl {
		//logger.Warning("[onProcessReport], nil pkg with transferID: ", transferID)
		return
	}

	for _, task := range dl.refTasks {
		task.UpdateDownloaderStatusHook(dl, detaSize/int64(len(dl.refTasks)), finishSize, totalSize)
	}
}

func (p *Service) finishDownloader(dl *Downloader, retCode int32) {
	logger.Infof("[finishDownloader] Download %v Finist with return Code %v", dl.ID, retCode)
	for _, task := range dl.refTasks {
		task.FinishDownloaderHook(dl, retCode)
	}

	dl.Finish()
}

func (p *Service) onTransferFinish(transferID string, retCode int32) {
	logger.Infof("[onTransferFinish] Download %v Finist with return Code %v", transferID, retCode)
	dl := QueryDownloader(transferID)
	if nil == dl {
		//	logger.Warning("[onProcessReport], nil pkg with transferID: ", transferID)
		return
	}
	p.finishDownloader(dl, retCode)
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
					//	logger.Warning("workTasks", len(p.workTasks), task.ID)
					time.Sleep(5 * time.Second)
				}
				p.workTasks[task.ID] = task
				logger.Warning("Start Single of task ", task.ID)
				waitNumber := task.WaitProcessNumber()
				logger.Warning("waitNumber", waitNumber)
				sendTaskStart := false
				if 0 == waitNumber {
					task.cancel(1, "Null task "+task.name)
				}
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

func (s *Service) downloaderDispatch() {
	for {
		select {
		case dl := <-s.downloadQueue:
			logger.Warning("start dl", dl.ID)
			err := dl.Start()
			if nil != err {
				s.finishDownloader(dl, TASK_FAILED)
			}
		}
	}
}
