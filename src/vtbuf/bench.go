package main

import (
	"flag"
	"fmt"
	"github.com/golang/protobuf/proto"
	"os"
	querypb "proto/queryresult"
	"queryresult/queryresult"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

var (
	benchmarks = sync.WaitGroup{}
	cpuprofile = flag.String("cpuprofile", "pprof", "write cpu profile to file")
	threads    = flag.Int("threads", 1, "number of threads")
	debug      = flag.Bool("debug", false, "debug mode")
	count      = flag.Int("count", 100000, "total counts for serialization/deserialization")
	format     = flag.String("format", "vtbuf", "serialization format: vtbuf, proto, bson")
)

func benchmark() {
	qr := queryresult.QueryResult{}
	for i := 0; i < 30; i++ {
		columnName := fmt.Sprintf("column-%d", i)
		qr.Fields = append(qr.Fields, queryresult.Field{Name: columnName, Type: 0})
	}
	qr.Rows = make([][]queryresult.Value, 3, 3)
	for i := 0; i < 3; i++ {
		for j := 0; j < 30; j++ {
			qr.Rows[i] = append(qr.Rows[i], queryresult.MakeString([]byte("abcdefghijklmnopqrstuvwxyz1234567890!@#$%^&*")))
		}
	}
	qr.RowsAffected = 3
	qr.InsertId = 101
	begin := time.Now()
	for i := 0; i < *count; i++ {
		switch *format {
		/*case "vtbuf":
		vtbuf := make([]byte, 1)
		vtbufqr := queryresult.QueryResult{}
		vtbufMarshal, _ := qr.MarshalVtbuf(vtbuf)
		vtbufqr.UnMarshalVtbuf(vtbufMarshal)
		if *debug {
			fmt.Printf("query result: %v\n", vtbufqr)
		}*/
		case "proto":
			qrpb := querypb.QueryResult{}
			queryresult.QueryResultToProto(&qr, &qrpb)
			pbbuf, _ := proto.Marshal(&qrpb)
			protoqr := mproto.QueryResult{}
			proto.Unmarshal(pbbuf, &qrpb)
			queryresult.ProtoToQueryResult(&qrpb, &protoqr)
			if *debug {
				fmt.Printf("query result: %v\n", protoqr)
			}
		/*case "bson":
		bsonbuf, _ := bson.Marshal(&qr)
		bsonqr := mproto.QueryResult{}
		bson.Unmarshal(bsonbuf, &bsonqr)
		if *debug {
			fmt.Printf("query result: %v\n", bsonqr)
		}*/
		default:
		}
	}
	cost := time.Now().Sub(begin)
	fmt.Printf("%d operations cost %d ns, %d ns per op\n", *count, cost.Nanoseconds(), cost.Nanoseconds()/int64(*count))
	benchmarks.Done()
}

func main() {
	google.Init()
	maxproc := 8
	if *threads*2 > maxproc {
		maxproc = *threads * 2
	}
	runtime.GOMAXPROCS(maxproc)
	f, err := os.Create(*cpuprofile)
	if err != nil {
		fmt.Printf("Cannot create cpu profile: %v\n", err)
		return
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	for i := 0; i < *threads; i++ {
		benchmarks.Add(1)
		go benchmark()
	}
	benchmarks.Wait()
}
