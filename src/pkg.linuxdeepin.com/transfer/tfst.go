package transfer

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"os"
	"sync"
)

type TransferSlice struct {
	Finish int64
	Begin  int64
	End    int64
}

type TransferStatus struct {
	statusFilePath string
	fileSize       int64
	blockSize      int64
	blockNum       int64
	blockStat      []TransferSlice

	writeLock sync.Mutex
}

func NewTransferStatus(filePath string, blockSize int64, fileSize int64) (*TransferStatus, error) {
	logger.Info("[NewTransferStatus] filePath, blockSize, fileSize", filePath, blockSize, fileSize)
	tfst := &TransferStatus{}
	tfst.statusFilePath = filePath
	blockNum := fileSize / blockSize
	if blockNum*blockSize < fileSize {
		blockNum += 1
	}
	tfst.fileSize = fileSize
	tfst.blockSize = blockSize
	tfst.blockNum = blockNum
	tfst.blockStat = make([]TransferSlice, blockNum)

	err := ioutil.WriteFile(tfst.statusFilePath, tfst.encode(), 0644)
	if err != nil {
		logger.Error("[NewTransferStatus] binary.Write failed:", err)
		return nil, err
	}
	return tfst, nil
}

func LoadTransferStatus(tfstFilePath string) (*TransferStatus, error) {
	buf, err := ioutil.ReadFile(tfstFilePath)
	if nil != err {
		return nil, err
	}

	tfst := &TransferStatus{}
	tfst.decode(buf)
	return tfst, nil
}

func (tfst *TransferStatus) Sync(index int64, slice TransferSlice) {
	tfst.writeLock.Lock()
	defer tfst.writeLock.Unlock()

	tfst.blockStat[index] = slice
	buf := tfst.encode()
	err := ioutil.WriteFile(tfst.statusFilePath, buf, 0644)
	if err != nil {
		logger.Error("[Sync] binary.Write failed:", err)
	}
}

func (tfst *TransferStatus) Remove() error {
	tfst.writeLock.Lock()
	defer tfst.writeLock.Unlock()
	logger.Info("[Remove]", tfst.statusFilePath)
	err := os.Remove(tfst.statusFilePath)
	return err
}

func (tfst *TransferStatus) encode() []byte {
	buf := new(bytes.Buffer)
	encoder := gob.NewEncoder(buf)
	encoder.Encode(tfst.statusFilePath)
	encoder.Encode(tfst.fileSize)
	encoder.Encode(tfst.blockSize)
	encoder.Encode(tfst.blockNum)
	for i, _ := range tfst.blockStat {
		err := encoder.Encode(tfst.blockStat[i])
		if nil != err {
			logger.Warning(err)
		}
	}
	return buf.Bytes()
}

func (tfst *TransferStatus) decode(buf []byte) {
	data := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(data)
	decoder.Decode(&tfst.statusFilePath)
	decoder.Decode(&tfst.fileSize)
	decoder.Decode(&tfst.blockSize)
	decoder.Decode(&tfst.blockNum)
	tfst.blockStat = make([]TransferSlice, tfst.blockNum)
	for i := int64(0); i < tfst.blockNum; i++ {
		err := decoder.Decode(&(tfst.blockStat[i]))
		if nil != err {
			logger.Warning(err)
		}
	}
}
