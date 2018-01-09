package gli

import (
	"fmt"
	"reflect"
)

type opt struct {
	names []string

	env      string
	defvalue string

	help        string
	placeholder string

	//

	pv       reflect.Value
	fieldIdx int
}

func (o opt) String() string {
	return fmt.Sprintf("opt{names=%v help=%v}", o.names, o.help)
}

func (o *opt) set(value string) {
	fv := o.pv.Field(o.fieldIdx)
	if fv.Type().Kind() == reflect.Ptr {
		pv := reflect.New(fv.Type().Elem())
		pv.Elem().Set(reflect.ValueOf(value))
		fv.Set(pv)
	} else {
		o.pv.Field(o.fieldIdx).Set(reflect.ValueOf(value))
	}
}
