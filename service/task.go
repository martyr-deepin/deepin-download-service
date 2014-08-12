package main

import (
	"fmt"
	"time"
)

type Task struct {
	ID     string
	name   string
	status TasKStatus
	//tranferid to pkg
	downloaders     map[string](*Downloader)
	waitDownloaders map[string](*Downloader)

	CB_Finish func(string)
	CB_Cancel func(string)
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

var _taskIDSeed = int64(0x0000)

func taskID() string {
	_taskIDSeed += 1
	return fmt.Sprintf("%v_task", _taskIDSeed)

}

func NewTask(name string, urls []string, sizes []int64, md5s []string, storeDir string) *Task {
	task := &Task{}
	task.downloaders = map[string](*Downloader){}
	task.waitDownloaders = map[string](*Downloader){}
	task.ID = taskID()
	var checkMD5 = true
	if len(urls) != len(md5s) {
		checkMD5 = false
	}
	var setSize = true
	if len(urls) != len(sizes) {
		setSize = false
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
	dl.totalSize = totalSize
}

func (p *Task) FinishDownloaderHook(dl *Downloader, retCode int32) {
	p.status.finished++
	logger.Warning(p.ID, p.status.finished, p.status.total, retCode)

	if 0 != retCode {
		logger.Warningf("Cancel Task %v", p.ID)
		p.Cancel()
		return
	}

	if p.status.finished == p.status.total {
		p.Finish()
		return
	}
}

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

func (p *Task) Start() error {
	for _, dl := range p.waitDownloaders {
		dl.Start()
	}
	p.waitDownloaders = map[string](*Downloader){}
	return nil
}

func (p *Task) WaitProcessNumber() int {
	return len(p.waitDownloaders)
}

func (p *Task) StartSingle() error {
	var startDL *Downloader
	for _, dl := range p.waitDownloaders {
		startDL = dl
		dl.Start()
		break
	}
	delete(p.waitDownloaders, startDL.ID)
	return nil
}

//Cancel task with error status
func (p *Task) Cancel() error {
	for _, dl := range p.downloaders {
		dl.UnRefTask(p)
	}
	if nil != p.CB_Cancel {
		go p.CB_Cancel(p.ID)
	}
	return nil
}

//Finish task sucess
func (p *Task) Finish() error {
	if nil != p.CB_Finish {
		go p.CB_Finish(p.ID)
	}
	return nil
}
