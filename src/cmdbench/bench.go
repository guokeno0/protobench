package main

import (
	"flag"
	"fmt"
	"github.com/golang/protobuf/proto"
	"os"
	"queryresult"
	querypb "queryresult/proto"
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
	format     = flag.String("format", "vtbuf", "serialization format: vtbuf, proto, directproto, allocproto")
)

func benchmark() {
	qr := queryresult.QueryResult{}
	qrpb := querypb.QueryResult{}
	for i := 0; i < 30; i++ {
		columnName := fmt.Sprintf("column-%d", i)
		qr.Fields = append(qr.Fields, queryresult.Field{Name: columnName, Type: 0})
		qrpb.Fields = append(qrpb.Fields, &querypb.Field{Name: proto.String(columnName), Type: querypb.Field_TYPE_STRING.Enum()})
	}
	qr.Rows = make([][]queryresult.Value, 3, 3)
	qrpb.Rows = make([]*querypb.Row, 3, 3)
	for i, _ := range qrpb.Rows {
		qrpb.Rows[i] = &querypb.Row{}
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 30; j++ {
			qr.Rows[i] = append(qr.Rows[i], queryresult.MakeString([]byte("abcdefghijklmnopqrstuvwxyz1234567890!@#$%^&*")))
			qrpb.Rows[i].Values = append(qrpb.Rows[i].Values, &querypb.Cell{Value: []byte("abcdefghijklmnopqrstuvwxyz1234567890!@#$%^&*")})
		}
	}
	qr.RowsAffected = 3
	qr.InsertId = 101
	qrpb.RowsAffected = proto.Uint64(3)
	qrpb.InsertId = proto.Uint64(101)
	begin := time.Now()
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	for i := 0; i < *count; i++ {
		switch *format {
		case "vtbuf":
			vtbuf := make([]byte, 1)
			vtbufqr := queryresult.QueryResult{}
			vtbufMarshal, _ := qr.MarshalVtbuf(vtbuf)
			vtbufqr.UnMarshalVtbuf(vtbufMarshal)
			if *debug {
				fmt.Printf("query result: %v\n", vtbufqr)
			}
		case "proto":
			qrpb := querypb.QueryResult{}
			queryresult.QueryResultToProto(&qr, &qrpb)
			pbbuf, _ := proto.Marshal(&qrpb)
			protoqr := queryresult.QueryResult{}
			proto.Unmarshal(pbbuf, &qrpb)
			queryresult.ProtoToQueryResult(&qrpb, &protoqr)
			if *debug {
				fmt.Printf("query result: %v\n", protoqr)
			}
		case "directproto":
			pbbuf, _ := proto.Marshal(&qrpb)
			newqrpb := querypb.QueryResult{}
			proto.Unmarshal(pbbuf, &newqrpb)
			if *debug {
				fmt.Printf("query result: %v\n", newqrpb)
			}
		case "allocproto":
			qrpb = querypb.QueryResult{}
			qrpb.Rows = make([]*querypb.Row, 3, 3)
			for i, _ := range qrpb.Rows {
				qrpb.Rows[i] = new(querypb.Row)
			}
			for i := 0; i < 3; i++ {
				for j := 0; j < 30; j++ {
					cell := new(querypb.Cell)
					cell.Value = []byte("abcdefghijklmnopqrstuvwxyz1234567890!@#$%^&*")
					qrpb.Rows[i].Values = append(qrpb.Rows[i].Values, cell)
				}
			}
			qrpb.RowsAffected = proto.Uint64(3)
			qrpb.InsertId = proto.Uint64(101)
		default:
		}
	}
	runtime.ReadMemStats(&m2)
	cost := time.Now().Sub(begin)
	fmt.Printf("%d operations cost %d ns, %d ns per op\n", *count, cost.Nanoseconds(), cost.Nanoseconds()/int64(*count))
	fmt.Printf("%d allocs allocating %d bytes, %d allocs/%d bytes per op\n",
		m2.Mallocs-m1.Mallocs,
		m2.TotalAlloc-m1.TotalAlloc,
		(m2.Mallocs-m1.Mallocs)/uint64(*count),
		(m2.TotalAlloc-m1.TotalAlloc)/uint64(*count))
	fmt.Printf("%d heap objects allocated, %d per op\n",
		m2.HeapObjects-m1.HeapObjects,
		(m2.HeapObjects-m1.HeapObjects)/uint64(*count))
	fmt.Printf("%d heap bytes allocated, %d bytes per op\n",
		m2.HeapAlloc-m1.HeapAlloc,
		(m2.HeapAlloc-m1.HeapAlloc)/uint64(*count))
	benchmarks.Done()
}

func main() {
	flag.Parse()
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
