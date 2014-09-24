package service

import (
	transfer "pkg.linuxdeepin.com/transfer"
)

var _transfer *transfer.Service

func GetTransfer() *transfer.Service {
	if nil == _transfer {
		_transfer = transfer.GetService()
	}
	return _transfer
}
