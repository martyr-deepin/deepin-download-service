.PHONY : test

CURDIR=$(shell pwd)
FIXGOPATH=$(CURDIR):$(GOPATH)
BIN_PATH=$(CURDIR)/bin

TRANSFER_SRC=$(CURDIR)/src/pkg.linuxdeepin.com/transfer
FTP_SRC =$(CURDIR)/src/pkg.linuxdeepin.com/transfer
SERVICE_SRC=$(CURDIR)/src/pkg.linuxdeepin.com/service
DDAEMON_SRC=$(CURDIR)/src/pkg.linuxdeepin.com/daemon

build:
	cd $(DDAEMON_SRC)  && GOPATH=$(FIXGOPATH) go build -o $(BIN_PATH)/deepin-download-service

test:
	cd $(TRANSFER_SRC) && GOPATH=$(FIXGOPATH) go test -v
	cd $(FTP_SRC) && GOPATH=$(FIXGOPATH) go test -v
	cd $(SERVICE_SRC)  && GOPATH=$(FIXGOPATH) go test -v

install:
	@mkdir -p $(DESTDIR)/usr/share/dbus-1/system-services
	@mkdir -p $(DESTDIR)/etc/dbus-1/system.d
	cp -a $(TRANSFER_SRC)/dbus/com.deepin.api.Transfer.service $(DESTDIR)/usr/share/dbus-1/system-services
	cp -a $(TRANSFER_SRC)/dbus/com.deepin.api.Transfer.conf $(DESTDIR)/etc/dbus-1/system.d/
	cp -a $(SERVICE_SRC)/dbus/com.deepin.download.service.service $(DESTDIR)/usr/share/dbus-1/system-services
	cp -a $(SERVICE_SRC)/dbus/com.deepin.download.service.conf $(DESTDIR)/etc/dbus-1/system.d/
	install -Dm755 $(BIN_PATH)/deepin-download-service $(DESTDIR)/usr/lib/deepin-daemon/deepin-download-service

clean:
	@-rm -rf bin/*
