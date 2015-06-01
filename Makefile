version = 1.0

all:
	go get
	go build -ldflags "-X main.globalVersionNumber $(version)"

clean:
	rm -f restatemachine

package: clean all
	./build_package.sh $(version)
