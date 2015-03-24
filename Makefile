all: dep proto bench

dep:
	go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

proto:
	protoc --go_out=./src/ ./proto/*.proto

bench:
	go install vtbuf
	mv bin/vtbuf bin/bench

