all:
	go build --ldflags "-s -w" .


clean:
	rm -rf md2html
