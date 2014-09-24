package transfer

import (
	"bytes"
	"testing"
)

func TestTransferStatusFile(t *testing.T) {
	tfst, err := NewTransferStatus(TmpDir+"/test.tfst", 1024, 8096)
	if nil != err {
		t.Error("Open tfst file failed")
	}
	tfst.blockStat[5].Begin = 0xFF
	tfst.Sync(1, tfst.blockStat[5])

	ntfst, err := LoadTransferStatus(TmpDir + "/test.tfst")
	if nil != err {
		t.Error("Open tfst file failed")
	}

	if !bytes.Equal(tfst.encode(), ntfst.encode()) {
		t.Error("Encode Status File Failed!")
	}
}
