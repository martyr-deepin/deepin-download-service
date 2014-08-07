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
	dlog "pkg.linuxdeepin.com/lib/log"
)

var logger = dlog.NewLogger("deepin-download-service")

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func main() {
	logger.Info("[main] deepin-download-service start")

	if !lib.UniqueOnSystem(DBUS_NAME) {
		logger.Warning("[main] There is aready a deepin-download-service running")
		os.Exit(0)
	}
	service := GetService()
	if err := dbus.InstallOnSystem(service); nil != err {
		logger.Error("[main]Install system bus failed", err)
		os.Exit(1)
	}

	logger.SetRestartCommand("/usr/lib/deepin-daemon/deepin-download-service", "--debug")
	if stringInSlice("-d", os.Args) || stringInSlice("--debug", os.Args) {
		logger.SetLogLevel(dlog.LEVEL_DEBUG)
	}

	dbus.DealWithUnhandledMessage()

	if err := dbus.Wait(); nil != err {
		os.Exit(1)
	}

	os.Exit(0)
}
