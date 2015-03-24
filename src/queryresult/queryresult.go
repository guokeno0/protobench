package queryresult

import (
	"encoding/binary"
	"fmt"
	"github.com/golang/protobuf/proto"
	querypb "queryresult/proto"
)

type Field struct {
	Name string
	Type int64
}

type QueryResult struct {
	Fields       []Field
	RowsAffected uint64
	InsertId     uint64
	Rows         [][]Value
}

type Value struct {
	Inner InnerValue
}

func (v Value) Raw() []byte {
	if v.Inner == nil {
		return nil
	}
	return v.Inner.raw()
}

type InnerValue interface {
	raw() []byte
	foo() int
	bar()
}

type String []byte

func (s String) raw() []byte {
	return []byte(s)
}

func (String) foo() int {
	return 0
}

func (String) bar() {
}

func MakeString(b []byte) Value {
	return Value{String(b)}
}

func RowToProto(values []Value) *querypb.Row {
	row := new(querypb.Row)
	row.Values = make([]*querypb.Cell, len(values))
	rowvalues := make([]querypb.Cell, len(values))
	for i, col := range values {
		rowvalues[i].Value = col.Raw()
		row.Values[i] = &rowvalues[i]
	}
	return row
}

// protobuf helpers for encoding/decoding

func QueryResultToProto(in *QueryResult, out *querypb.QueryResult) {
	out.RowsAffected = proto.Uint64(in.RowsAffected)
	out.InsertId = proto.Uint64(in.InsertId)

	out.Rows = make([]*querypb.Row, len(in.Rows))
	for i, row := range in.Rows {
		out.Rows[i] = RowToProto(row)
	}

	out.Fields = make([]*querypb.Field, len(in.Fields))
	for i, field := range in.Fields {
		pfield := &querypb.Field{
			Name: proto.String(field.Name),
			Type: querypb.Field_Type(field.Type).Enum(),
		}
		out.Fields[i] = pfield
	}
}

func ProtoToQueryResult(in *querypb.QueryResult, out *QueryResult) {
	out.RowsAffected = in.GetRowsAffected()
	out.InsertId = in.GetInsertId()

	out.Fields = make([]Field, len(in.Fields))
	for i, f := range in.Fields {
		out.Fields[i].Name = f.GetName()
		out.Fields[i].Type = int64(f.GetType())
	}

	out.Rows = make([][]Value, len(in.Rows))
	for i, r := range in.Rows {
		row := make([]Value, len(r.Values))
		for j, val := range r.Values {
			if val.Value == nil {
				row[j] = Value{}
			} else {
				row[j] = MakeString(val.Value)
			}
		}
		out.Rows[i] = row
	}
}

// vtbuf encoding/decoding
const (
	SIZEUINT64 = 8
)

// VTbuf format
// RowsAffected : uint64
// InsertId : uint64
// number_of_fields : uint64
// Fields: {field_name_start : uint64, field_name_end : uint64, field_type : uint64}[]
// number_of_rows: uint64
// Rows: {num_of_cells: uint64}
// Cells: {data_start : uint64, data_end : uint64}
// field_name_data
// cell_data
//
// A cell with empty cell_type is the Row deliminator

func vtbufWriteUint64(buf []byte, val uint64) []byte {
	binary.LittleEndian.PutUint64(buf, val)
	return buf[SIZEUINT64:]
}

func vtbufWriteBytes(buf []byte, val []byte) []byte {
	copy(buf, val)
	return buf[len(val):]
}

func (queryResult *QueryResult) MarshalVtbuf(buf []byte) (retbuf []byte, size uint64) {
	bufsize := uint64(0)
	// Calculate length of field data
	for _, f := range queryResult.Fields {
		bufsize += uint64(len(f.Name))
	}
	// Calculate number of cells and length of cell data
	cells := uint64(0)
	for _, row := range queryResult.Rows {
		cells += uint64(len(row))
		for _, cell := range row {
			bufsize += uint64(len(cell.Raw()))
		}
	}
	// Calculate the offset for data bytes
	dataoff := uint64(SIZEUINT64 + // RowsAffected
		SIZEUINT64 + // InsertId
		SIZEUINT64 + // number_of_fields
		(SIZEUINT64+SIZEUINT64+SIZEUINT64)*len(queryResult.Fields) + // Fields
		SIZEUINT64 + // number_of_rows
		SIZEUINT64*len(queryResult.Rows) + // Rows
		(SIZEUINT64+SIZEUINT64)*int(cells)) // Cells
	bufsize += dataoff
	if bufsize > uint64(cap(buf)) {
		buf = make([]byte, bufsize, bufsize)
	}
	retbuf = buf
	databuf := buf[dataoff:]
	// RowsAffected
	buf = vtbufWriteUint64(buf, queryResult.RowsAffected)
	// InsertId
	buf = vtbufWriteUint64(buf, queryResult.InsertId)
	// number_of_fields
	buf = vtbufWriteUint64(buf, uint64(len(queryResult.Fields)))
	// Fields and Fields name data
	for _, field := range queryResult.Fields {
		// field_name_start
		buf = vtbufWriteUint64(buf, dataoff)
		// field_name_data
		databuf = vtbufWriteBytes(databuf, []byte(field.Name))
		// field_name_end
		dataoff += uint64(len(field.Name))
		buf = vtbufWriteUint64(buf, dataoff)
		// field_type
		buf = vtbufWriteUint64(buf, uint64(field.Type))
	}
	// number_of_rows
	buf = vtbufWriteUint64(buf, uint64(len(queryResult.Rows)))
	// Rows
	for _, row := range queryResult.Rows {
		buf = vtbufWriteUint64(buf, uint64(len(row)))
	}
	// Cells and cell data
	for _, row := range queryResult.Rows {
		for _, cell := range row {
			// data_start
			buf = vtbufWriteUint64(buf, dataoff)
			if cell.Raw() != nil {
				// cell_data
				databuf = vtbufWriteBytes(databuf, cell.Raw())
				dataoff += uint64(len(cell.Raw()))
			}
			// data_end
			buf = vtbufWriteUint64(buf, dataoff)
		}
	}
	return retbuf, bufsize
}

func vtbufReadUint64(buf []byte) (err error, val uint64, retbuf []byte) {
	if cap(buf) <= SIZEUINT64 {
		return fmt.Errorf("buffer underflow"), 0, buf
	}
	val = binary.LittleEndian.Uint64(buf)
	return nil, val, buf[SIZEUINT64:]
}

func (queryResult *QueryResult) UnMarshalVtbuf(buf []byte) (err error) {
	origbuf := buf
	// RowsAffected
	err, rowsAffected, buf := vtbufReadUint64(buf)
	if err != nil {
		return err
	}
	queryResult.RowsAffected = rowsAffected
	// InsertId
	err, insertId, buf := vtbufReadUint64(buf)
	if err != nil {
		return err
	}
	queryResult.InsertId = insertId
	// number_of_fields
	err, fields, buf := vtbufReadUint64(buf)
	if err != nil {
		return err
	}
	queryResult.Fields = make([]Field, fields)

	// Fields
	for f := 0; f < int(fields); f++ {
		var fns, fne, ft uint64
		// field_name_start
		err, fns, buf = vtbufReadUint64(buf)
		if err != nil {
			return err
		}
		// field_name_end
		err, fne, buf = vtbufReadUint64(buf)
		if err != nil {
			return err
		}
		// field_type
		err, ft, buf = vtbufReadUint64(buf)
		if err != nil {
			return err
		}
		if (fns < fne) && fns < uint64(cap(origbuf)) && fne <= uint64(cap(origbuf)) {
			queryResult.Fields[f].Name = string(origbuf[fns:fne])
			queryResult.Fields[f].Type = int64(ft)
			continue
		}
		return fmt.Errorf("invalid field encoding: %ld %ld %ld", fns, fne, ft)
	}
	// number_of_rows
	err, rows, buf := vtbufReadUint64(buf)
	if err != nil {
		return err
	}
	// Rows
	queryResult.Rows = make([][]Value, rows, rows)
	for r := 0; r < int(rows); r++ {
		var cells uint64
		err, cells, buf = vtbufReadUint64(buf)
		if err != nil {
			return err
		}
		queryResult.Rows[r] = make([]Value, cells, cells)
	}
	// Cells
	for _, r := range queryResult.Rows {
		for i, _ := range r {
			var ds, de uint64
			// data_start
			err, ds, buf = vtbufReadUint64(buf)
			if err != nil {
				return err
			}
			// data_end
			err, de, buf = vtbufReadUint64(buf)
			if err != nil {
				return err
			}
			if ds == de {
				r[i] = Value{}
				continue
			}
			if ds < de && ds < uint64(cap(origbuf)) && de <= uint64(cap(origbuf)) {
				r[i] = MakeString(origbuf[ds:de])
				continue
			}
			return fmt.Errorf("Invalid cell encoding: %ld %ld", ds, de)
		}
	}
	return nil
}
