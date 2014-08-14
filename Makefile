CURDIR=$(shell pwd)
all:
	cd transfer && go build -o transfer
	-rm -rf transfer/dbus-factory/go
	cd transfer/dbus-factory/in.json && ../../build/json_build.py
	#fix go src path
	@mkdir -p transfer/dbus-factory/go/src/dbus
	@mv transfer/dbus-factory/go/src/com transfer/dbus-factory/go/src/dbus
	cd service && GOPATH=$(CURDIR)/transfer/dbus-factory/go go build -o deepin-download-service
	cd service/dbus-factory/in.json && ../../build/json_build.py

install:
	@mkdir -p $(DESTDIR)/usr/share/dbus-1/system-services
	@mkdir -p $(DESTDIR)/etc/dbus-1/system.d
	install -Dm755 transfer/transfer $(DESTDIR)/usr/lib/deepin-api/transfer
	cp -a transfer/dbus/com.deepin.api.Transfer.service $(DESTDIR)/usr/share/dbus-1/system-services
	cp -a transfer/dbus/com.deepin.api.Transfer.conf $(DESTDIR)/etc/dbus-1/system.d/
	install -Dm755 service/deepin-download-service $(DESTDIR)/usr/lib/deepin-daemon/deepin-download-service
	cp -a service/dbus/com.deepin.download.service.service $(DESTDIR)/usr/share/dbus-1/system-services
	cp -a service/dbus/com.deepin.download.service.conf $(DESTDIR)/etc/dbus-1/system.d/

clean:
	@-rm -rf transfer/dbus-factory/go
	@-rm -rf transfer/dbus-factory/qml
	@-rm -rf transfer/transfer
	@-rm -rf service/dbus-factory/go
	@-rm -rf service/dbus-factory/qml
	@-rm -rf service/deepin-download-service
