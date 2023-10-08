package gli

import (
	"reflect"
)

type option struct {
	names []string

	env      string
	defValue string
	defDesc  string

	required bool
	assigned bool

	help        string
	placeholder string

	ownerV   reflect.Value
	fieldIdx int

	nondefFirstParsing bool
}

func (o option) longestName() string {
	maxlen := -1
	var maxname string
	for _, n := range o.names {
		nlen := len(n)
		if nlen > maxlen {
			maxlen = nlen
			maxname = n
		}
	}

	return maxname
}

func (o *option) setValue(value interface{}) error {
	o.ownerV.Elem().Field(o.fieldIdx).Set(reflect.ValueOf(value))
	o.assigned = true
	return nil
}
