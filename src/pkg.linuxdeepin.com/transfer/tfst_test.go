package main

import (
	"bytes"
	"fmt"
	"testing"
)

func Test_TransferStatusFile(t *testing.T) {
	tfst, _ := NewTransferStatus("/tmp/test.tfst", 1024, 8096)
	tfst.blockStat[5].Begin = 0xFF
	tfst.Sync(1, tfst.blockStat[5])
	fmt.Println(tfst)

	ntfst, _ := LoadTransferStatus("/tmp/test.tfst")
	fmt.Println(ntfst)

	if !bytes.Equal(tfst.encode(), ntfst.encode()) {
		t.Error("Encode Status File Failed!")
	}
}
