package main

import (
	transferAPI "dbus/com/deepin/api/transfer"
)

const (
	TRANSFER_NAME = "com.deepin.api.Transfer"
	TRANSFER_PATH = "/com/deepin/api/Transfer"
)

var _transferDBus *transferAPI.Transfer

func TransferDbus() *transferAPI.Transfer {
	if nil == _transferDBus {
		var err error
		_transferDBus, err = transferAPI.NewTransfer(TRANSFER_NAME, TRANSFER_PATH)
		if nil != err {
			panic("[init] Connect com.deepin.api.Transfer Failed")
		}
	}
	return _transferDBus
}
