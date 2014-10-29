package transfer

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"pkg.linuxdeepin.com/ftp"
)

const (
	UnlockData = int32(0x09)
)

const (
	CacheSize          = 8096
	ErrorRetryWaitTime = 100
)

type FtpClient struct {
	supportRange bool
	username     string
	password     string
	key          string
	c            *ftp.ServerConn

	dataLock sync.Mutex
	cmdLock  sync.Mutex
}

func (p *FtpClient) SupportRange() bool {
	return p.supportRange
}

func (p *FtpClient) QuerySize(url string) (int64, error) {
	logger.Warning("QuerySize")
	p.lockCmd()
	defer p.unlockCmd()

	remotePath := strings.Replace(url, "ftp://", "", -1)
	remotePath = remotePath[len(strings.Split(remotePath, "/")[0])+1 : len(remotePath)]

	return p.c.Size(remotePath)
}

func (p *FtpClient) NewRequest(url string) (Request, error) {
	remotePath := strings.Replace(url, "ftp://", "", -1)
	remotePath = remotePath[len(strings.Split(remotePath, "/")[0])+1 : len(remotePath)]

	request := &FtpRequest{}
	request.url = url
	request.remotePath = remotePath
	request.client = p
	return request, nil
}

type FtpRequest struct {
	url        string
	remotePath string
	size       int64
	client     *FtpClient

	RequestBase
}

func (r *FtpRequest) QuerySize() (int64, error) {
	return r.client.QuerySize(r.url)
}

func (r *FtpRequest) Download(localFilePath string) error {
	logger.Infof("[Download] %v", localFilePath)

	var err error
	r.client.lockData()
	defer r.client.unlockData()
	r.client.lockCmd()
	defer r.client.unlockCmd()

	defer func() {
		if nil != err {
			logger.Warning("Login(err)")
			r.client.Login()
		}
	}()

	dlfile, err := os.OpenFile(localFilePath, os.O_CREATE|os.O_RDWR, DefaultFileMode)
	defer dlfile.Close()
	if err != nil {
		logger.Error(err)
		return err
	}

	data, err := r.client.c.Retr(r.remotePath)
	if err != nil {
		logger.Error(err, r.remotePath)
		return err
	}
	defer data.Close()

	capacity := CacheSize
	writtenBytes := int64(0)
	buf := make([]byte, 0, capacity)
	for {
		if TaskCancel == r.statusCheck() {
			return fmt.Errorf("Download Cancel")
		}
		m, e := data.Read(buf[len(buf):cap(buf)])
		buf = buf[0 : len(buf)+m]

		if nil != r.progress {
			r.progress(int64(m), writtenBytes+int64(len(buf)), r.size)
		}

		if len(buf) == cap(buf) {
			dlfile.WriteAt(buf, writtenBytes)
			dlfile.Sync()
			writtenBytes += int64(len(buf))
			buf = make([]byte, 0, capacity)
			continue
		}

		if e == io.EOF {
			dlfile.WriteAt(buf, writtenBytes)
			dlfile.Sync()
			logger.Warning("Read Buffer End with", e)
			break
		}
		if nil != e {
			logger.Error("Ftp Download Error: ", e)
			time.Sleep(ErrorRetryWaitTime * time.Millisecond)
			break
		}
	}
	return nil
}

func (r *FtpRequest) DownloadRange(begin int64, end int64) ([]byte, error) {
	logger.Infof("[DownloadRange] %v-%v", begin, end)

	var err error
	r.client.lockData()
	defer r.client.unlockData()
	r.client.lockCmd()
	defer r.client.unlockCmd()

	defer func() {
		if nil != err {
			logger.Warning("Login(err)")
			r.client.Login()
		}
	}()

	if 0 != begin {
		logger.Warning(begin, "Rest to")
		err = r.client.c.Rest(begin)
		if nil != err {
			logger.Error(err)
			return nil, err
		}
	}
	data, err := r.client.c.Retr(r.remotePath)
	if err != nil {
		logger.Error(err, r.remotePath)
		return nil, err
	}
	data.Close()

	capacity := end - begin
	buf := make([]byte, 0, capacity)
	for {
		if TaskCancel == r.statusCheck() {
			return nil, fmt.Errorf("Download Cancel")
		}
		m, e := data.Read(buf[len(buf):cap(buf)])
		buf = buf[0 : len(buf)+m]

		if nil != r.progress {
			r.progress(int64(m), begin+int64(len(buf)), r.size)
		}

		if e == io.EOF {
			logger.Warning(e)
			break
		}
		if nil != e {
			logger.Info("Read e: ", e)
			time.Sleep(ErrorRetryWaitTime * time.Millisecond)
			return nil, e
		}
	}

	return buf, nil
}

func quitAllFtpClient() {
	for _, c := range _ftpClientPool {
		c.lockCmd()
		c.c.Quit()
		c.unlockCmd()
	}
	_ftpClientPool = map[string](*FtpClient){}
}

func GetFtpClient(username string, password string, addr string) (*FtpClient, error) {
	var err error
	if !strings.Contains(addr, ":") {
		addr = addr + ":21"
	}
	key := username + password + addr
	client := _ftpClientPool[key]
	if nil == client {
		client = &FtpClient{}
		client.username = username
		client.password = password
		client.c, err = ftp.Connect(addr)
		if err != nil {
			logger.Error(err)
			return nil, fmt.Errorf("Create Ftp Connect Failed")
		}
		err = client.c.Login(username, password)
		if err != nil {
			logger.Error(err)
			return nil, fmt.Errorf("Login Ftp Server Failed")
		}

		client.dataLock = sync.Mutex{}
		client.cmdLock = sync.Mutex{}
		client.key = key
		_ftpClientPool[key] = client
	}

	return client, nil
}

func (p *FtpClient) Login() {
	//		p.c.Logout()
	err := p.c.Login(p.username, p.password)
	if err != nil {
		logger.Error(err)
		//return nil, fmt.Errorf("Login Ftp Server Failed")
	}
}

func (p *FtpClient) lockCmd() {
	logger.Warning("Lock Cmd")
	p.cmdLock.Lock()
}
func (p *FtpClient) unlockCmd() {
	logger.Warning("Unlock Cmd")
	p.cmdLock.Unlock()
}

func (p *FtpClient) lockData() {
	logger.Warning("Lock Data")
	p.dataLock.Lock()
}

func (p *FtpClient) unlockData() {
	logger.Warning("Unlock Data")
	p.dataLock.Unlock()
}

func (p *FtpClient) Download(t *Transfer, ftpPath string) error {
	p.lockData()
	defer p.unlockData()

	r, err := p.c.Retr(ftpPath)
	if err != nil {
		logger.Error(err, ftpPath)
		p.c.Quit()
		delete(_ftpClientPool, p.key)
		return fmt.Errorf("Download Cancel")
	}

	capacity := t.fileSize + 512
	buf := make([]byte, 0, capacity*2)
	for {
		//checkTaskStatus will block if status is pause
		if TaskCancel == t.Status() {
			return fmt.Errorf("Download Cancel")
		}
		m, e := r.Read(buf[len(buf):cap(buf)])
		buf = buf[0 : len(buf)+m]
		if len(buf) == cap(buf) {
			newBuf := make([]byte, cap(buf)*2)
			copy(newBuf, buf[:len(buf)])
			buf = newBuf[:len(buf)]
			//logger.Warning("extern", cap(buf), buf)
		}
		t.detaSize += int64(m)
		t.downloadSize = 0 + int64(len(buf))
		t.totalSize = t.fileSize
		GetService().sendProcessReportSignal(t.ID, int64(m), int64(len(buf)), t.fileSize)

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
	dlfile, err := os.OpenFile(t.localFile, os.O_CREATE|os.O_RDWR, DefaultFileMode)
	defer dlfile.Close()
	if err != nil {
		logger.Error(err)
		return err
	}
	dlfile.Write(buf)
	dlfile.Sync()
	return nil
}

var _ftpClientPool map[string](*FtpClient)

func init() {
	_ftpClientPool = map[string](*FtpClient){}
}
