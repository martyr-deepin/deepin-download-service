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

	"pkg.linuxdeepin.com/lib/dbus"
	"pkg.linuxdeepin.com/service"
	"pkg.linuxdeepin.com/transfer"
)

func main() {
	err := transfer.LoadDBus()
	if nil != err {
		os.Exit(1)
	}

	err = service.LoadDBus()
	if nil != err {
		os.Exit(1)
	}

	dbus.DealWithUnhandledMessage()

	if err := dbus.Wait(); nil != err {
		os.Exit(1)
	}

	os.Exit(0)
}
