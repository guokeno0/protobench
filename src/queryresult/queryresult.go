package queryresult

import querypb "proto/queryresult"

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

func (String) foo() []byte {
	return nil
}

func (String) bar() []byte {
	return nil
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
