BINARY ?= gsplash
GO ?= go
GOFLAGS ?=
LDFLAGS ?= -s -w
BUILD_DIR ?= build
DESTDIR ?=
PREFIX ?= /usr
BINDIR ?= $(PREFIX)/bin
DATADIR ?= $(PREFIX)/share/gsplash
SYSTEMD_UNITDIR ?= /lib/systemd/system

.PHONY: all build clean install

all: build

build: $(BUILD_DIR)/$(BINARY)

$(BUILD_DIR)/$(BINARY): main.go fb/fb.go go.mod
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GO) build $(GOFLAGS) -trimpath -ldflags "$(LDFLAGS)" -o $@ .

install: $(BUILD_DIR)/$(BINARY)
	install -d $(DESTDIR)$(BINDIR)
	install -m 0755 $(BUILD_DIR)/$(BINARY) $(DESTDIR)$(BINDIR)/$(BINARY)
	install -d $(DESTDIR)$(DATADIR)
	install -m 0644 assets/gsplash.png $(DESTDIR)$(DATADIR)/gsplash.png
	install -d $(DESTDIR)$(SYSTEMD_UNITDIR)
	install -m 0644 systemd/gsplash.service $(DESTDIR)$(SYSTEMD_UNITDIR)/gsplash.service

clean:
	rm -rf $(BUILD_DIR)
