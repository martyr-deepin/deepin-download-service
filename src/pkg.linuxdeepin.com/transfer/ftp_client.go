package main

import (
	"io"
	"os"
	"strings"
	"time"

	"./ftp"
)

type FtpClient struct {
	key   string
	tasks map[int32](*TranferTaskInfo)
	c     *ftp.ServerConn
}

func (p *FtpClient) Download(taskinfo *TranferTaskInfo, ftpPath string) error {
	for {
		if 0 != len(p.tasks) {
			time.Sleep(1 * time.Second)
		} else {
			p.tasks[taskinfo.taskid] = taskinfo
			break
		}
	}
	defer delete(p.tasks, taskinfo.taskid)

	r, err := p.c.Retr(ftpPath)
	if err != nil {
		logger.Error(err, ftpPath)
		p.c.Quit()
		delete(_connectPool, p.key)
		return TransferError("Download Cancel")
	}

	capacity := taskinfo.fileSize + 512
	buf := make([]byte, 0, capacity*2)
	for {
		//checkTaskStatus will block if status is pause
		if TASK_ST_CANCEL == checkTaskStatus(taskinfo) {
			return TransferError("Download Cancel")
		}
		m, e := r.Read(buf[len(buf):cap(buf)])
		buf = buf[0 : len(buf)+m]
		if len(buf) == cap(buf) {
			newBuf := make([]byte, cap(buf)*2)
			copy(newBuf, buf[:len(buf)])
			buf = newBuf[:len(buf)]
			//logger.Warning("extern", cap(buf), buf)
		}
		taskinfo.detaSize += int64(m)
		taskinfo.downloadSize = 0 + int64(len(buf))
		taskinfo.totalSize = taskinfo.fileSize
		GetTransfer().ProcessReport(taskinfo.taskid, int64(m), int64(len(buf)), taskinfo.fileSize)

		if e == io.EOF {
			logger.Warning(e)
			//logger.Info("Read io.EOF: ", len(buf))
			break
		}
		if e != nil {
			time.Sleep(4 * time.Millisecond)
			logger.Info("Read e: ", e)
			break
		}
	}

	r.Close()
	//write to file
	dlfile, err := os.OpenFile(taskinfo.localFile, os.O_CREATE|os.O_RDWR, 0755)
	defer dlfile.Close()
	if err != nil {
		logger.Error(err)
		return err
	}
	dlfile.Write(buf)
	dlfile.Sync()
	return nil
}

var _connectPool map[string](*FtpClient)

func init() {
	_connectPool = map[string](*FtpClient){}
}

func GetFtpClient(username string, password string, addr string) (*FtpClient, error) {
	if !strings.Contains(addr, ":") {
		addr = addr + ":21"
	}
	var err error
	key := username + password + addr
	client := _connectPool[key]
	if nil == client {
		client = &FtpClient{}
		client.c, err = ftp.Connect(addr)
		if err != nil {
			logger.Error(err)
			return nil, TransferError("Create Ftp Connect Failed")
		}
		err = client.c.Login(username, password)
		if err != nil {
			logger.Error(err)
			return nil, TransferError("Login Ftp Server Failed")
		}
		client.tasks = map[int32](*TranferTaskInfo){}
		client.key = key
		_connectPool[key] = client
	}

	return client, nil
}

func doFtpDownload(taskinfo *TranferTaskInfo) error {
	ftpUrl := taskinfo.url[6:len(taskinfo.url)]
	addr := strings.Split(ftpUrl, "/")[0]
	ftpPath := ftpUrl[len(addr)+1 : len(ftpUrl)]

	client, err := GetFtpClient("anonymous", "anonymous", addr)
	if err != nil {
		logger.Error(err)
		return TransferError("Download Cancel")
	}

	return client.Download(taskinfo, ftpPath)
}
