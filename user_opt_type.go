package gli

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Parsable represents an string->YOURTYPE convertible
type Parsable interface {
	Parse(str string) error
}

////////////////////////////////////////////////////////////////////////////////

// Date is a parsable type of date.
//
// see https://golang.org/pkg/time/#ParseDuration
type Date time.Time

func (d *Date) Parse(str string) error {
	tm, err := time.Parse("2006-01-02", str)
	if err != nil {
		tm, err = time.Parse("2006/01/02", str)
		if err != nil {
			return err
		}
	}

	*d = Date(tm)

	return nil
}

func (d Date) Time() time.Time {
	return time.Time(d)
}

////////////////////////////////////////////////////////////////////////////////

// Duration is a parsable type of time duration.
//
// see https://golang.org/pkg/time/#ParseDuration
type Duration time.Duration

func (d *Duration) Parse(str string) error {
	dur, err := time.ParseDuration(str)
	if err != nil {
		return err
	}

	*d = Duration(dur)

	return nil
}

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

///////////////////////////////////////////////////////////////////////////////

// Range is a parsable type of min:max.
type Range struct {
	Min, Max string
}

func (r *Range) Parse(str string) error {
	if !strings.Contains(str, ":") {
		return fmt.Errorf("range format is min:max")
	}

	min := str[:strings.Index(str, ":")]
	max := str[strings.Index(str, ":")+1:]

	r.Min, r.Max = min, max

	return nil
}

///////////////////////////////////////////////////////////////////////////////

// StrList is a parsable type of ["s", "t", "r"]
type StrList []string

func (l *StrList) Parse(str string) error {
	*l = (*l)[:0]
	list := strings.Split(str, ",")
	for i := 0; i < len(list); i++ {
		*l = append(*l, strings.TrimSpace(list[i]))
	}

	return nil
}

func (l StrList) Contains(s string) bool {
	for _, v := range l {
		if v == s {
			return true
		}
	}
	return false
}

///////////////////////////////////////////////////////////////////////////////

// IntList is a parsable type of [1, 2, 3]
type IntList []int

func (l *IntList) Parse(str string) error {
	*l = (*l)[:0]
	list := strings.Split(str, ",")
	for i := 0; i < len(list); i++ {
		s := strings.TrimSpace(list[i])
		n, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return err
		}
		*l = append(*l, int(n))
	}

	return nil
}

func (l IntList) Contains(i int) bool {
	for _, v := range l {
		if v == i {
			return true
		}
	}
	return false
}

///////////////////////////////////////////////////////////////////////////////

// Map is a parsable type of key=value or key:value pair.
type Map map[string]string

func (m *Map) Parse(str string) error {
	if *m == nil {
		*m = make(map[string]string)
	}

	for _, s := range strings.Split(str, ",") {
		s = strings.TrimSpace(s)
		pos := strings.Index(s, ":")
		if pos == -1 {
			return errors.New("no separator in Map")
		}

		key, value := s[:pos], s[pos+1:]
		if value == "" {
			delete(*m, key)
		} else {
			(*m)[key] = value
		}
	}

	return nil
}
