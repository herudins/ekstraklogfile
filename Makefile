outname ?= ekstraklog
outnamecomp ?= $(outname)_compressed

run: 
	go run -ldflags "-s -w" .

build: clean build-linux build-windows

build-linux:
	@printf "%s\n" "Build binary file for linux..."
	@go clean
	CGO_ENABLED=0 GOOS="linux" GOARCH="amd64" go build -o bin/$(outname) -ldflags "-s -w" -tags=nomsgpack

build-windows:
	@printf "%s\n" "Build binary file for windows..."
	@go clean
	CGO_ENABLED=0 GOOS="windows" GOARCH="amd64" go build -o bin/$(outname).exe -ldflags "-s -w" -tags=nomsgpack

clean:
	rm -f bin/$(outname)
	rm -f bin/$(outname).exe
