package transfer

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"pkg.deepin.io/ftp"
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
	addr         string
	key          string
	c            *ftp.ServerConn

	err      error
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

	defer p.ErrorRecover()

	remotePath := strings.Replace(url, "ftp://", "", -1)
	remotePath = remotePath[len(strings.Split(remotePath, "/")[0])+1 : len(remotePath)]

	if nil == p.c {
		err := fmt.Errorf("nil Ftp Connect")
		p.err = err
		return 0, err
	}
	size, err := p.c.Size(remotePath)
	p.err = err
	return size, err
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
	logger.Infof("Download %v", localFilePath)

	var err error
	r.client.lockData()
	defer r.client.unlockData()
	r.client.lockCmd()
	defer r.client.unlockCmd()

	defer r.client.ErrorRecover()

	dlfile, err := os.OpenFile(localFilePath, os.O_CREATE|os.O_RDWR, DefaultFileMode)
	defer dlfile.Close()
	if err != nil {
		logger.Error("OpenFile %v Failed: %v", localFilePath, err)
		return err
	}

	if nil == r.client.c {
		err := fmt.Errorf("nil Ftp Connect")
		r.client.err = err
		return err
	}

	data, err := r.client.c.Retr(r.remotePath)
	if err != nil {
		r.client.err = fmt.Errorf("Retr %v Failed: %v", r.remotePath, err)
		logger.Error(r.client.err)
		return r.client.err
	}
	defer data.Close()

	logger.Info("Try to Read Data of ", r.url)
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
			logger.Info("Read Buffer End with", e)
			break
		}
		if nil != e {
			r.client.err = fmt.Errorf("Read Ftp Data Failed: %v", e)
			logger.Error(r.client.err)
			return r.client.err
		}
	}
	return nil
}

func (r *FtpRequest) DownloadRange(begin int64, end int64) ([]byte, error) {
	logger.Infof("DownloadRange %v-%v", begin, end)

	var err error
	r.client.lockData()
	defer r.client.unlockData()
	r.client.lockCmd()
	defer r.client.unlockCmd()

	defer r.client.ErrorRecover()

	if nil == r.client.c {
		err := fmt.Errorf("nil Ftp Connect")
		r.client.err = err
		return nil, err
	}

	if 0 != begin {
		logger.Infof("Rest to %v", begin)
		err = r.client.c.Rest(begin)
		if nil != err {
			r.client.err = fmt.Errorf("Reset %v Failed: %v", r.url, err)
			logger.Error(r.client.err)
			return nil, r.client.err
		}
	} else {
		logger.Infof("Download From %v", begin)
	}

	data, err := r.client.c.Retr(r.remotePath)
	if err != nil {
		r.client.err = fmt.Errorf("Retr %v Failed: %v", r.url, err)
		logger.Error(r.client.err)
		return nil, r.client.err
	}
	defer data.Close()

	logger.Info("Try to Read Data of ", r.url)
	capacity := end - begin
	buf := make([]byte, 0, capacity)
	for {
		if TaskCancel == r.statusCheck() {
			return nil, fmt.Errorf("Cancel Download: %v ", r.url)
		}
		m, e := data.Read(buf[len(buf):cap(buf)])
		buf = buf[0 : len(buf)+m]

		if nil != r.progress {
			r.progress(int64(m), begin+int64(len(buf)), r.size)
		}

		if e == io.EOF {
			logger.Info("Download Read Buffer End with", e)
			break
		}
		if nil != e {
			r.client.err = fmt.Errorf("Download %v Error: %v", r.url, e)
			logger.Error(r.client.err)
			return nil, r.client.err
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

var _clientLock sync.Mutex

func GetFtpClient(username string, password string, addr string) (*FtpClient, error) {
	logger.Infof("Lock GetFtpClient of %v", addr)
	_clientLock.Lock()
	defer _clientLock.Unlock()
	defer logger.Infof("Unlock GetFtpClient of %v", addr)

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
		client.addr = addr

		defer client.ErrorRecover()
		client.c, err = ftp.Connect(addr)
		if err != nil {
			client.err = fmt.Errorf("Create Ftp Connect Failed: %v", err)
			logger.Error(client.err)
			return nil, client.err
		}
		err = client.c.Login(username, password)
		if err != nil {
			client.err = fmt.Errorf("Login Ftp Server Failed: %v", err)
			logger.Error(client.err)
			return nil, client.err
		}

		client.dataLock = sync.Mutex{}
		client.cmdLock = sync.Mutex{}
		client.key = key
		_ftpClientPool[key] = client
	}

	return client, nil
}

func (p *FtpClient) Login() error {
	//		p.c.Logout()
	err := p.c.Login(p.username, p.password)
	if err != nil {
		logger.Error(err)
		return fmt.Errorf("Login Ftp Server Failed")
	}
	return nil
}

func (p *FtpClient) ErrorRecover() {
	if nil == p.err {
		return
	}
	p.err = nil
	p.Reset()
}

func (p *FtpClient) Reset() error {
	var err error
	if nil != p.c {
		p.c.Logout()
		p.c.Quit()
	}
	logger.Warningf("Reset FtpClient, Wait 10s...")
	time.Sleep(10 * time.Second)

	p.c, err = ftp.Connect(p.addr)
	if err != nil {
		err = fmt.Errorf("Connect to %v Failed: %v", p.addr, err)
		logger.Error(err)
		return err
	}

	err = p.c.Login(p.username, p.password)
	if err != nil {
		err = fmt.Errorf("Login Ftp Server Failed: %v", err)
		logger.Error(err)
		return err
	}
	fmt.Println("Reset Success", p.c)
	return nil
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

var _ftpClientPool map[string](*FtpClient)

func init() {
	_ftpClientPool = map[string](*FtpClient){}
}
