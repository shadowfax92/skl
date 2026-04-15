PREFIX ?= $(HOME)/bin
VERSION ?= 0.1.0

build:
	go build -ldflags "-X skl/cmd.Version=$(VERSION)" -o skl .

install: build
	cp skl $(PREFIX)/skl
	codesign --force --sign - $(PREFIX)/skl

clean:
	rm -f skl

.PHONY: build install clean
