CURDIR=$(shell pwd)

BIN_PATH=$(CURDIR)/bin

TRANSFER_SRC=$(CURDIR)/src/pkg.linuxdeepin.com/transfer
SERVICE_SRC=$(CURDIR)/src/pkg.linuxdeepin.com/service

all:
	cd $(TRANSFER_SRC) && go build -o $(BIN_PATH)/transfer
	cd $(SERVICE_SRC)  && go build -o $(BIN_PATH)/deepin-download-service

install:
	@mkdir -p $(DESTDIR)/usr/share/dbus-1/system-services
	@mkdir -p $(DESTDIR)/etc/dbus-1/system.d
	install -Dm755 $(BIN_PATH)/transfer $(DESTDIR)/usr/lib/deepin-api/transfer
	cp -a $(TRANSFER_SRC)/dbus/com.deepin.api.Transfer.service $(DESTDIR)/usr/share/dbus-1/system-services
	cp -a $(TRANSFER_SRC)/dbus/com.deepin.api.Transfer.conf $(DESTDIR)/etc/dbus-1/system.d/
	install -Dm755 $(BIN_PATH)/deepin-download-service $(DESTDIR)/usr/lib/deepin-daemon/deepin-download-service
	cp -a $(SERVICE_SRC)/dbus/com.deepin.download.service.service $(DESTDIR)/usr/share/dbus-1/system-services
	cp -a $(SERVICE_SRC)/dbus/com.deepin.download.service.conf $(DESTDIR)/etc/dbus-1/system.d/

clean:
	@-rm -rf bin/* 
