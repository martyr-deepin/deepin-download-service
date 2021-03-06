/*
Copyright (C) 2011~2014 Deepin, Inc.
              2011~2014 He Li

Author:     He Li <me@iceyer.net>
Maintainer: He Li <me@iceyer.net>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package main

import (
	"os"
	"time"

	"pkg.deepin.io/lib/dbus"
	dlogger "pkg.deepin.io/lib/log"
	"pkg.deepin.io/service"
	"pkg.deepin.io/transfer"
)

const (
	DaemonExitTime = 60 //Second
)

var logger = dlogger.NewLogger("dde-api/transfer/daemon")

func main() {
	err := transfer.LoadDBus()
	if nil != err {
		logger.Error(err)
		os.Exit(1)
	}

	err = service.LoadDBus()
	if nil != err {
		logger.Error(err)
		os.Exit(1)
	}

	go startDaemon()

	dbus.DealWithUnhandledMessage()

	if err := dbus.Wait(); nil != err {
		logger.Error(err)
		os.Exit(1)
	}
	logger.Error("daemon Exit")
	os.Exit(0)
}

var daemonTimer *time.Timer

func startDaemon() {
	daemonTimer = time.NewTimer(DaemonExitTime * time.Second)
	for {
		select {
		case <-daemonTimer.C:
			if (transfer.GetTransferManager().TransferCount() == 0) && (service.GetService().TaskCount() == 0) {
				transfer.GetTransferManager().Close()
				service.GetService().Close()
				os.Exit(0)
			}
			daemonTimer.Reset(DaemonExitTime * time.Second)
		}
	}
}
