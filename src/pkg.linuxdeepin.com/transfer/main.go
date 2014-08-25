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

	"pkg.linuxdeepin.com/lib"
	"pkg.linuxdeepin.com/lib/dbus"
	dlogger "pkg.linuxdeepin.com/lib/log"
)

var logger = dlogger.NewLogger("dde-api/transfer")

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func main() {
	defer logger.EndTracing()
	logger.Info("[main] Start Transfer Service")
	if !lib.UniqueOnSystem(TRANSFER_DEST) {
		logger.Warning("[main] There already has an Transfer daemon running.")
		return
	}

	// configure logger
	logger.SetRestartCommand("/usr/lib/deepin-api/transfer", "--debug")
	if stringInSlice("-d", os.Args) || stringInSlice("--debug", os.Args) {
		logger.SetLogLevel(dlogger.LevelDebug)
	}

	transfer := GetTransfer()

	err := dbus.InstallOnSystem(transfer)
	if err != nil {
		logger.Error("[main] InstallOnSystem Error", err)
		panic(err)
	}

	dbus.DealWithUnhandledMessage()

	if err = dbus.Wait(); err != nil {
		logger.Error("[main] Lost dbus session:", err)
		os.Exit(1)
	}

	os.Exit(0)
}
