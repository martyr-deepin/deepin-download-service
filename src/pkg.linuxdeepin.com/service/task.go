package service

import (
	"time"

	"pkg.linuxdeepin.com/lib/utils"
)

const (
	TaskVaild   = true
	TaskInVaild = false
)

type Task struct {
	ID      string
	name    string
	isVaild bool
	status  TasKStatus
	//tranferid to pkg
	downloaders     map[string](*Downloader)
	waitDownloaders map[string](*Downloader)
	workDownloaders map[string](*Downloader)

	CB_Finish func(string)
	CB_Cancel func(string, int32, string)
}

type TasKStatus struct {
	status int32

	progress     int32
	totalSize    int64
	downloadSize int64
	finished     int32
	total        int32

	//speed statistics
	speedStater SpeedStater
}

type SpeedStater struct {
	Rate int64

	lastBits        int64
	lastTime        time.Time
	lastIndex       int
	speedStat       [10]int64
	sumHistorySpeed int64
}

func (s *SpeedStater) Refresh() {
	now := time.Now()
	deta := now.Sub(s.lastTime).Nanoseconds()
	//计算过去10s的平均速度
	//lastBytes
	speed := s.lastBits * 1000 * 1000 * 1000 / deta
	index := s.lastIndex
	s.sumHistorySpeed = s.sumHistorySpeed - s.speedStat[index] + speed
	s.speedStat[index] = speed
	s.lastIndex = (index + 1) % 10
	s.Rate = s.sumHistorySpeed / 10
	s.lastBits = 0
	s.lastTime = now
}

var _taskIDSeed = int64(0x0000)

func taskID() string {
	return utils.GenUuid() + "_task"
}

func NewTask(name string, urls []string, sizes []int64, md5s []string, storeDir string) *Task {
	task := &Task{}
	task.isVaild = TaskVaild
	task.downloaders = map[string](*Downloader){}
	task.waitDownloaders = map[string](*Downloader){}
	task.workDownloaders = map[string](*Downloader){}
	task.ID = taskID()
	var checkMD5 = true
	if len(urls) != len(md5s) {
		checkMD5 = false
	}
	var setSize = true
	if len(urls) != len(sizes) {
		setSize = false
	}

	if 0 == len(urls) {
		logger.Error("[NewTask] Empty url list ", task.name, task.ID)
	}

	//TODO: delete the same urls
	for i, url := range urls {
		md5 := ""
		if checkMD5 {
			md5 = md5s[i]
		}
		size := int64(0)
		if setSize {
			size = sizes[i]
		}

		dl := GetDownloader(url, size, md5, storeDir, "")

		task.downloaders[dl.ID] = dl
		task.waitDownloaders[dl.ID] = dl
		dl.RefTask(task)
	}
	task.status.total = int32(len(task.downloaders))
	return task
}

func (p *Task) querySize() error {
	p.status.totalSize = 0
	for _, dl := range p.downloaders {
		if 0 != dl.totalSize {
			p.status.totalSize += dl.totalSize
		} else {
			p.status.totalSize += dl.QuerySize()
		}
	}
	return nil
}

func (p *Task) UpdateDownloaderStatusHook(dl *Downloader, currentSize int64, downloadSize int64, totalSize int64) {
	p.status.speedStater.lastBits += currentSize
	dl.downloadSize = downloadSize
	if 0 == dl.totalSize {
		dl.totalSize = totalSize
	}
}

func (p *Task) FinishDownloaderHook(dl *Downloader, retCode int32) {
	p.status.finished++
	logger.Warning(p.ID, p.status.finished, p.status.total, retCode)

	if TASK_SUCCESS != retCode {
		logger.Warningf("Cancel Task %v", p.ID)
		p.cancel(retCode, "Download Package Error")
		return
	}

	if p.status.finished == p.status.total {
		p.finish()
		return
	}
}

//RefresStatus will return the progress status of task
func (p *Task) RefreshStatus() (int32, int32, int32, int32, int64, int64) {
	p.status.downloadSize = 0
	p.status.totalSize = 0
	for _, dl := range p.downloaders {
		p.status.downloadSize += dl.downloadSize
		p.status.totalSize += dl.totalSize
	}
	progress := int64(0)
	if 0 != p.status.totalSize {
		progress = p.status.downloadSize * 100 / p.status.totalSize
	}
	p.status.speedStater.Refresh()
	return int32(progress), int32(p.status.speedStater.Rate),
		int32(p.status.finished), int32(p.status.total),
		p.status.downloadSize, p.status.totalSize
}

//Cancel task with error status
func (p *Task) cancel(errCode int32, errStr string) error {
	for _, dl := range p.downloaders {
		dl.UnRefTask(p)
	}
	if nil != p.CB_Cancel {
		go p.CB_Cancel(p.ID, errCode, errStr)
	}
	p.clear()
	return nil
}

//Finish task sucess
func (p *Task) finish() error {
	if nil != p.CB_Finish {
		go p.CB_Finish(p.ID)
	}
	p.clear()
	return nil
}

func (p *Task) clear() {
	p.isVaild = TaskInVaild
	p.waitDownloaders = map[string](*Downloader){}
	p.workDownloaders = map[string](*Downloader){}
}

//Start will start all wait download
func (p *Task) Start() error {
	for _, dl := range p.waitDownloaders {
		dl.Start()
	}
	p.waitDownloaders = map[string](*Downloader){}
	p.workDownloaders = p.downloaders
	return nil
}

func (p *Task) WaitProcessNumber() int {
	return len(p.waitDownloaders)
}

//StartSingle will start one download each call
func (p *Task) StartSingle() *Downloader {
	for _, dl := range p.waitDownloaders {
		p.workDownloaders[dl.ID] = dl
		delete(p.waitDownloaders, dl.ID)
		return dl
	}
	return nil
}
func (p *Task) Vaild() bool {
	return p.isVaild
}

func (p *Task) Pause() {
	for _, dl := range p.workDownloaders {
		dl.Pause()
	}
}

func (p *Task) Resume() {
	for _, dl := range p.workDownloaders {
		dl.Resume()
	}
}
func (p *Task) Stop() {
	for _, dl := range p.workDownloaders {
		dl.UnRefTask(p)
	}
	p.clear()
}
