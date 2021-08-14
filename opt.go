package gli

import (
	"fmt"
	"reflect"
)

type option struct {
	Names []string

	Env      string
	DefValue string
	DefDesc  string

	Required bool
	Assigned bool

	Help        string
	Placeholder string

	OwnerV   reflect.Value
	fieldIdx int

	nondefFirstParsing bool
}

func (o option) LongestName() string {
	maxlen := -1
	var maxname string
	for _, n := range o.Names {
		nlen := len(n)
		if nlen > maxlen {
			maxlen = nlen
			maxname = n
		}
	}

	return maxname
}

func (o option) String() string {
	return fmt.Sprintf("option{Names:%v}", o.Names)
}

func (o *option) SetValue(value interface{}) error {
	o.OwnerV.Elem().Field(o.fieldIdx).Set(reflect.ValueOf(value))
	o.Assigned = true
	return nil
}
