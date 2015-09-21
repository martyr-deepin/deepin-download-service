.PHONY : test

CURDIR=$(shell pwd)
FIXGOPATH=$(CURDIR):$(GOPATH)
BIN_PATH=$(CURDIR)/bin

SRC_PATH=$(CURDIR)/src/pkg.deepin.io
TRANSFER_SRC=$(SRC_PATH)/transfer
FTP_SRC =$(SRC_PATH)/transfer
SERVICE_SRC=$(SRC_PATH)/service
DDAEMON_SRC=$(SRC_PATH)/daemon

ifndef USE_GCCGO
    GOBUILD = go build
else
    LDFLAGS = $(shell pkg-config --libs gio-2.0)
    GOBUILD = go build -compiler gccgo -gccgoflags "${LDFLAGS}"
endif

build:
	cd $(DDAEMON_SRC)  && GOPATH=$(FIXGOPATH) ${GOBUILD} -o $(BIN_PATH)/deepin-download-service

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
