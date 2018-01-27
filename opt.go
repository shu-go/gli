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
