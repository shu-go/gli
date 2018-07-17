package gli

import (
	"fmt"
	"reflect"
)

type option struct {
	Names []string
	Kind  reflect.Kind

	Env      string
	DefValue string

	Help        string
	Placeholder string

	OwnerV   reflect.Value
	fieldIdx int
}

func (o option) String() string {
	return fmt.Sprintf("option{Names:%v}", o.Names)
}

func (o *option) SetValue(value interface{}) error {
	o.OwnerV.Elem().Field(o.fieldIdx).Set(reflect.ValueOf(value))
	return nil
}
