# How to use this to benchmark protobuf:
# To download:
jeffjiang@jeffjiang2:~$ git clone https://github.com/guokeno0/protobench.git

# To compile:
jeffjiang@jeffjiang2:~$ cd protobench/

jeffjiang@jeffjiang2:~/protobench$ make

export GOPATH=`pwd`; \

        go get -u github.com/golang/protobuf/{proto,protoc-gen-go}
        
protoc --go_out=./src/queryresult/ ./proto/queryresult.proto

export GOPATH=`pwd`; \

        go install cmdbench; \

        mv bin/cmdbench bin/bench

jeffjiang@jeffjiang2:~/protobench$ ./bin/bench --help

Usage of ./bin/bench:

  -count=100000: total counts for serialization/deserialization

  -cpuprofile="pprof": write cpu profile to file

  -debug=false: debug mode

  -format="vtbuf": serialization format: vtbuf, proto

  -threads=1: number of threads

# To run the benchmark:
## Test protobuf
jeffjiang@jeffjiang2:~/protobench$ ./bin/bench -count=1000000 -format="proto" -threads=4

1000000 operations cost 455539055429 ns, 455539 ns per op

1000000 operations cost 455589428734 ns, 455589 ns per op

1000000 operations cost 455727697932 ns, 455727 ns per op

1000000 operations cost 455841313457 ns, 455841 ns per op
## Test vtbuf, as a hit about how much space for improvement
jeffjiang@jeffjiang2:~/protobench$ ./bin/bench -count=1000000 -format="vtbuf" -threads=4

1000000 operations cost 145815346835 ns, 145815 ns per op

1000000 operations cost 145974197197 ns, 145974 ns per op

1000000 operations cost 146131051738 ns, 146131 ns per op

1000000 operations cost 146155638429 ns, 146155 ns per op
