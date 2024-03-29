package gli

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// TypeDecoder is used to convert command-line arguments to optional values.
// It decodes a string to another type of value.
//
// # To be decoded
//
// If you declare your cli interface like:
//
//	type myCLIRootCommand struct {
//	    Opt1 []string `cli:"opt1"`
//	}
//
// And end-user gives a command-line like:
//
//	my.exe  --opt1 a,b,c
//
// The library gli finds that the opt1 is of type []string, and tries to decode the string "a,b,c" into []string.
//
// gli finds a decoder by calling [LookupTypeDecoder].
// And then gli calls the decoder function to decode "a,b,c" into []string{"a","b","c"}.
// Finally, gli assigns []string{"a","b","c"} to opt1.
//
// # Implemented types (enabled by default)
//
//   - (built-in types of golang)
//   - time.Time (local time; --opt yyyy/mm/dd or --opt yyyy-mm-dd)
//   - time.Duration
//   - []string (--opt a,b,c)
//   - []int (--opt 1,2,3)
//   - map[string]string (--opt key:value,key:value)
//   - gli.Range{Min,Max string} (--opt min:max)
//
// # User defined types
//
//  1. Define a decoder function as TypeDecoder
//  2. Call [gli.RegisterTypeDecoder](reflect.TypeOf(anyValueOfTheType), decoderFunc)
//
// # TypeDecoder
//
// s is a string to decode.
//
// tag is a StructTag of the option.
//
// If firstTime is true, the function should reset v and then decode s into v.
// Otherwise, it simply decodes s into v.
// This parameter is useful when appending the contents of a value.
type TypeDecoder func(s string, v reflect.Value, tag reflect.StructTag, firstTime bool) error

// See [TypeDecoder].
func RegisterTypeDecoder(typ reflect.Type, dec TypeDecoder) {
	typRegistry.Register(typ, dec)
}

func LookupTypeDecoder(typ reflect.Type) TypeDecoder {
	return typRegistry.Lookup(typ)
}

type Range struct {
	Min, Max string
}

////////////////////////////////////////////////////////////////////////////////

type typeRegistry struct {
	m   sync.Mutex
	reg map[reflect.Type]TypeDecoder
}

func (t *typeRegistry) Register(typ reflect.Type, dec TypeDecoder) {
	t.m.Lock()
	t.reg[typ] = dec
	t.m.Unlock()
}

func (t *typeRegistry) Lookup(typ reflect.Type) TypeDecoder {
	t.m.Lock()
	dec, found := t.reg[typ]
	t.m.Unlock()

	if !found {
		return nil
	}
	return dec
}

////////////////////////////////////////////////////////////////////////////////

func timeDecoder(s string, v reflect.Value, tag reflect.StructTag, firstTime bool) error {
	tm, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		tm, err = time.ParseInLocation("2006/01/02", s, time.Local)
		if err != nil {
			return err
		}
	}
	v.Set(reflect.ValueOf(tm))
	return nil
}

func durationDecoder(s string, v reflect.Value, tag reflect.StructTag, firstTime bool) error {
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	v.Set(reflect.ValueOf(dur))
	return nil
}

var commaRE = regexp.MustCompile(`(?:\\,|[^,])+`)

func strSliceDecoder(s string, v reflect.Value, tag reflect.StructTag, firstTime bool) error {
	if firstTime {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	}
	slist := commaRE.FindAllString(s, -1)
	v.Set(reflect.AppendSlice(v, reflect.ValueOf(slist)))
	return nil
}

func intSliceDecoder(s string, v reflect.Value, tag reflect.StructTag, firstTime bool) error {
	if firstTime {
		v.Set(reflect.MakeSlice(v.Type(), 0, 0))
	}

	ilist := []int{}
	for _, elem := range commaRE.FindAllString(s, -1) {
		elem = strings.TrimSpace(elem)
		elem = strings.ReplaceAll(elem, `\,`, `,`)
		n, err := strconv.ParseInt(elem, 10, 0)
		if err != nil {
			return err
		}
		ilist = append(ilist, int(n))
	}
	v.Set(reflect.AppendSlice(v, reflect.ValueOf(ilist)))
	return nil
}

func strMapDecoder(s string, v reflect.Value, tag reflect.StructTag, firstTime bool) error {
	if firstTime || v.IsNil() {
		v.Set(reflect.MakeMap(v.Type()))
	}

	for _, elem := range commaRE.FindAllString(s, -1) {
		elem = strings.TrimSpace(elem)
		elem = strings.ReplaceAll(elem, `\,`, `,`)
		pos := strings.Index(elem, ":")
		if pos == -1 {
			return errors.New("no separator in Map")
		}

		key, value := elem[:pos], elem[pos+1:]
		if value == "" {
			v.SetMapIndex(reflect.ValueOf(key), reflect.Value{})
		} else {
			v.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
	}
	return nil
}

func strRangeDecoder(s string, v reflect.Value, tag reflect.StructTag, firstTime bool) error {
	if !strings.Contains(s, ":") {
		return fmt.Errorf("range format is min:max")
	}

	r := Range{}
	min := s[:strings.Index(s, ":")]
	max := s[strings.Index(s, ":")+1:]
	r.Min, r.Max = min, max

	v.Set(reflect.ValueOf(r))

	return nil
}

////////////////////////////////////////////////////////////////////////////////

var typRegistry typeRegistry

func init() {
	typRegistry = typeRegistry{
		reg: make(map[reflect.Type]TypeDecoder),
	}

	RegisterTypeDecoder(reflect.TypeOf(time.Time{}), timeDecoder)
	RegisterTypeDecoder(reflect.TypeOf(time.Duration(0)), durationDecoder)
	RegisterTypeDecoder(reflect.TypeOf([]string{}), strSliceDecoder)
	RegisterTypeDecoder(reflect.TypeOf([]int{}), intSliceDecoder)
	RegisterTypeDecoder(reflect.TypeOf(map[string]string{}), strMapDecoder)

	RegisterTypeDecoder(reflect.TypeOf(Range{}), strRangeDecoder)
}
