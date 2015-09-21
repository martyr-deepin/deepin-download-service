package service

import (
	transfer "pkg.deepin.io/transfer"
)

var _transfer *transfer.TransferManager

func GetTransfer() *transfer.TransferManager {
	if nil == _transfer {
		_transfer = transfer.GetTransferManager()
	}
	return _transfer
}
