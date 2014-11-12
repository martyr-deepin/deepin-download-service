package service

import (
	transfer "pkg.linuxdeepin.com/transfer"
)

var _transfer *transfer.TransferManager

func GetTransfer() *transfer.TransferManager {
	if nil == _transfer {
		_transfer = transfer.GetTransferManager()
	}
	return _transfer
}
