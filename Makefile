BINDDIR := /usr/bin

build:
	dep ensure
	./build.sh
all: build
install:
	mkdir -p ${DESTDIR}${BINDDIR}
	cp build/linux/x86_64/lpmx ${DESTDIR}${BINDDIR}/

