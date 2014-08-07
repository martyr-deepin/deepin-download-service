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
	"errors"
	"fmt"
	"strings"
	"time"

	apitransfer "dbus/com/deepin/api/transfer"

	"pkg.linuxdeepin.com/lib/dbus"
)

const (
	DBUS_NAME = "com.deepin.download.service"
	DBUS_PATH = "/com/deepin/download/service"
	DBUS_IFC  = "com.deepin.download.service"
)

const (
	TRANSFER_NAME = "com.deepin.api.Transfer"
	TRANSFER_PATH = "/com/deepin/api/Transfer"
)
const (
	C_FREE_PROCESS = int32(0x40)
)

/*
	App/Task is list of debian packages
	Pkg is mean single debian package
*/
type Pkg struct {
	fileName     string
	size         int64
	downloadSize int64
	url          string
}

const (
	TS_FINISH = int32(0x10)
)

type Task struct {
	id     string
	name   string
	status int32
	//package
	pkgs        []string
	unstartPkgs []string
	processPkgs []string
	finishPkgs  []string

	//speed statistics
	speedStater SpeedStater

	//tranferid to pkg
	transferList []Pkg
	pkgTransfer  map[int32]*Pkg

	downloadSize int64
	totalSize    int64
	//storeDir
	storeDir string
}

type SpeedStater struct {
	lastBytes    int64
	lastTime     time.Time
	speedStat    [10]int64
	historySpeed int64
	avagerSpeed  int64
	index        int
}

type Service struct {
	idSeed    int64              //seed to generate taskid(string)
	tasks     map[string](*Task) //taskid to task
	transfers map[int32]string   //tranferID to taskid

	//control the max gocontinue to download
	maxProcess  int32
	curProcess  int32
	freeProcess chan int32

	transferDbus *apitransfer.Transfer

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

func GetUrlFileName(url string) string {
	list := strings.Split(url, "/")
	return list[len(list)-1]
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
	now := time.Now()
	for taskid, task := range p.tasks {
		if TS_FINISH == task.status {
			continue
		}
		total := len(task.pkgs)
		finish := len(task.finishPkgs)
		progress := int64(0)
		if 0 != task.totalSize {
			progress = task.downloadSize * 100 / task.totalSize
		}
		deta := now.Sub(task.speedStater.lastTime).Nanoseconds()
		//计算过去10s的平均速度
		//lastBytes
		speed := task.speedStater.lastBytes * 1000 * 1000 * 1000 / deta
		index := task.speedStater.index
		task.speedStater.historySpeed = (task.speedStater.historySpeed*10 - task.speedStater.speedStat[index] + speed) / 10
		task.speedStater.speedStat[index] = speed
		task.speedStater.index = (index + 1) % 10
		curSpeed := task.speedStater.historySpeed
		task.speedStater.lastBytes = 0
		task.speedStater.lastTime = now

		logger.Info(taskid, progress, finish, total, task.downloadSize, task.totalSize, curSpeed/1024, "KByte/s")
		p.Update(taskid, int32(progress), int32(curSpeed), int32(finish), int32(total), task.downloadSize, task.totalSize)
	}
	timer.Reset(1 * time.Second)
}

func (p *Service) init() {
	logger.Info("[init] Init Service")
	t, err := apitransfer.NewTransfer(TRANSFER_NAME, TRANSFER_PATH)
	if nil != err {
		p.errorHandle("", E_INIT_TRANSFER_API, err)
		panic("[init]Connect com.deepin.api.Transfer Failed")
	}
	t.ConnectFinshReport(p.onPkgFinish)
	t.ConnectProcessReport(p.onProcessReport)
	p.transferDbus = t
	p.curProcess = 0
	p.maxProcess = 8
	p.freeProcess = make(chan int32, p.maxProcess+1)
	p.tasks = map[string](*Task){}
	p.transfers = map[int32]string{}

	go p.startUpdateTaskInfoTimer()
}

func (p *Service) onProcessReport(transferID int32, detaBytes int64, finishBytes int64, totalBytes int64) {
	taskid := p.transfers[transferID]
	task := p.tasks[taskid]
	if nil == task {
		return
	}
	task.downloadSize = finishBytes
	task.speedStater.lastBytes += detaBytes

}

func (p *Service) onPkgFinish(transferID int32, retCode int32) {
	logger.Infof("[onPkgFinish] Download %v Finist with return Code %v", transferID, retCode)
	taskid := p.transfers[transferID]
	task := p.tasks[taskid]
	if nil == task {
		logger.Warning("[onPkgFinish], nil taskid with transferID: ", transferID)
		return
	}
	pkg := task.pkgTransfer[transferID]
	if nil == pkg {
		logger.Warning("[onPkgFinish], nil pkg with transferID: ", transferID)
		return
	}
	task.finishPkgs = append(task.finishPkgs, pkg.fileName)

	p.freeProcess <- C_FREE_PROCESS
	p.curProcess = p.curProcess - 1
	finish := len(task.finishPkgs)
	total := len(task.pkgs)

	if finish == total {
		task.status = TS_FINISH
		p.Update(taskid, int32(100), int32(task.speedStater.historySpeed), int32(finish), int32(total), task.downloadSize, task.totalSize)
		p.Finish(taskid)
	}

	if 0 != retCode {
		p.errorHandle(taskid, 0, errors.New("Download "))
		//if error, stop task
		p.StopTask(taskid)
	}
}

//AddTask will add download task to transfer queue and return
//Task is list of debian packages
//pkg is mean single debian package
func (p *Service) AddTask(taskName string, urls []string, storeDir string) (taskid string) {
	logger.Infof("[AddTask] %v", taskName)
	task := &Task{}
	task.name = taskName
	task.storeDir = storeDir
	task.pkgTransfer = map[int32](*Pkg){}
	task.speedStater.lastTime = time.Now()
	task.speedStater.lastBytes = 0
	task.speedStater.index = 0
	for _, url := range urls {
		pkg := Pkg{}
		pkg.url = url
		pkg.downloadSize = 0
		pkgName := GetUrlFileName(url)
		logger.Infof("Add pkg: %v", pkgName)
		pkg.fileName = pkgName
		task.pkgs = append(task.pkgs, pkgName)
		task.transferList = append(task.transferList, pkg)
	}
	task.unstartPkgs = task.pkgs
	taskid = p.genTaskID()
	task.id = taskid
	p.tasks[taskid] = task
	go p.startTask(task)
	return taskid
}

//PauseTask will pause Task
func (p *Service) PauseTask(taskid string) {
	task := p.tasks[taskid]
	if nil == task {
		logger.Info("Error taskid")
		p.errorHandle(taskid, E_INVAILD_TASKID, errors.New("Invaid Taskid"))
		return
	}

	for transferID, _ := range task.pkgTransfer {
		logger.Info("Pause task: ", transferID)
		p.transferDbus.Pause(transferID)
	}
	p.Pause(taskid)
}

//ResumTask will Resume Task
func (p *Service) ResumeTask(taskid string) {
	task := p.tasks[taskid]
	if nil == task {
		logger.Info("Error taskid")
		p.errorHandle(taskid, E_INVAILD_TASKID, errors.New("Invaid Taskid"))
		return
	}
	logger.Info(task.pkgTransfer)
	for transferID, _ := range task.pkgTransfer {
		logger.Info("Resume task: ", transferID)
		ret, err := p.transferDbus.Resume(transferID)
		if nil != err {
			logger.Fatal(err, ret)
		}
	}
	p.Resume(taskid)
}

//StopTask will stop Task and DELETE Task
func (p *Service) StopTask(taskid string) {
	task := p.tasks[taskid]
	if nil == task {
		logger.Info("Error taskid")
		p.errorHandle(taskid, E_INVAILD_TASKID, errors.New("Invaid Taskid"))
		return
	}

	for transferID, _ := range task.pkgTransfer {
		logger.Info("Cancel task: ", transferID)
		p.transferDbus.Cancel(transferID)
	}

	for transferID, _ := range task.pkgTransfer {
		delete(p.transfers, transferID)
	}
	delete(p.tasks, taskid)
	p.Stop(taskid)
}

const (
	E_INIT_TRANSFER_API = int32(0x80)
	E_DOWNLOAD_PKG      = E_INIT_TRANSFER_API + 1
	E_INVAILD_TASKID    = E_INIT_TRANSFER_API + 2
)

func (p *Service) genTaskID() string {
	p.idSeed = p.idSeed + 1
	return fmt.Sprintf("%v", p.idSeed)
}

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

func (p *Service) queryTaskInfo(task *Task) {
	for _, t := range task.transferList {
		t.size, _ = p.transferDbus.QuerySize(t.url)
		task.totalSize += t.size
	}
}

func (p *Service) startTask(task *Task) {
	logger.Infof("[startTask] %v", task)
	p.queryTaskInfo(task)
	p.Start(task.id)
	for _, t := range task.transferList {
		logger.Infof("[] Process %v/%v", p.curProcess, p.maxProcess)
		if p.curProcess >= p.maxProcess {
			p.waitProcess()
		}
		logger.Infof("[] Process %v/%v", p.curProcess, p.maxProcess)
		transferID, err := p.transferDbus.Download(t.url, task.storeDir+"/"+t.fileName, 0)
		if nil != err {
			p.errorHandle(task.id, E_DOWNLOAD_PKG, err)
			p.Stop(task.id)
			return
		}
		task.pkgTransfer[transferID] = &t
		p.transfers[transferID] = task.id
		task.processPkgs = append(task.processPkgs, t.fileName)
		p.curProcess++
	}
}

func (p *Service) errorHandle(taskid string, retCode int32, err error) {
	logger.Warningf("handle Error: %v", err)
	p.Error(taskid, retCode, err.Error())
}
