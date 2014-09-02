package main

import (
	"bytes"
	"encoding/gob"
	"io/ioutil"
	"os"
	"sync"
)

type TransferSlice struct {
	finish int64
	begin  int64
	end    int64
}

type TransferStatus struct {
	statusFilePath string
	statusFile     *os.File
	fileSize       int64
	blockSize      int64
	blockNum       int64
	blockStat      []TransferSlice

	writeLock sync.Mutex
}

func NewTransferStatus(filePath string, blockSize int64, fileSize int64) (*TransferStatus, error) {
	logger.Info("NewTransferStatus filePath, blockSize, fileSize", filePath, blockSize, fileSize)
	var err error
	tfst := &TransferStatus{}
	tfst.statusFilePath = filePath
	tfst.statusFile, err = os.Create(tfst.statusFilePath)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	blockNum := fileSize / blockSize
	if blockNum*blockSize < fileSize {
		blockNum += 1
	}
	tfst.fileSize = fileSize
	tfst.blockSize = blockSize
	tfst.blockNum = blockNum
	tfst.blockStat = make([]TransferSlice, blockNum)

	_, err = tfst.statusFile.Write(tfst.encode())
	if err != nil {
		logger.Error("binary.Write failed:", err)
		return nil, err
	}
	tfst.statusFile.Sync()
	return tfst, nil
}

func LoadTransferStatus(tfstFilePath string) (*TransferStatus, error) {
	buf, err := ioutil.ReadFile(tfstFilePath)
	if nil != err {
		return nil, err
	}

	tfst := &TransferStatus{}
	tfst.statusFile, err = os.Open(tfstFilePath)
	if nil != err {
		return nil, err
	}
	tfst.decode(buf)
	return tfst, nil
}

func (tfst *TransferStatus) Sync(index int64, slice TransferSlice) {
	tfst.writeLock.Lock()
	defer tfst.writeLock.Unlock()

	tfst.blockStat[index] = slice
	tfst.statusFile.WriteAt(tfst.encode(), 0)
	tfst.statusFile.Sync()
}

func (tfst *TransferStatus) Close() {
	tfst.writeLock.Lock()
	defer tfst.writeLock.Unlock()
	tfst.statusFile.Close()
}

func (tfst *TransferStatus) Remove() error {
	tfst.writeLock.Lock()
	defer tfst.writeLock.Unlock()
	tfst.statusFile.Close()
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
	for _, slice := range tfst.blockStat {
		encoder.Encode(slice)
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
		decoder.Decode(&(tfst.blockStat[i]))
	}
}
