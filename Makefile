all: dep bench

dep:
	go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

protodef:
	protoc --go_out=./src/queryresult/ ./proto/queryresult.proto

bench: protodef
	go install cmdbench
	mv bin/cmdbench bin/bench

