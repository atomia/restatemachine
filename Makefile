version = 1.1

ifndef GOPATH
	export GOPATH=$(shell pwd)/gopath
endif

all:
	go get github.com/tools/godep
	$(GOPATH)/bin/godep restore
	go build -o restatemachine -ldflags "-X main.globalVersionNumber $(version)"

clean:
	rm -f *.deb *.rpm
	rm -f restatemachine
	rm -rf gopath

package: clean all
	./build_package.sh $(version)
