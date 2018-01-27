package gli

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Parsable interface {
	Parse(str string) error
}

////////////////////////////////////////////////////////////////////////////////

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

type StrList []string

func (l *StrList) Parse(str string) error {
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

type IntList []int

func (l *IntList) Parse(str string) error {
	list := strings.Split(str, ",")
	for i := 0; i < len(list); i++ {
		s := strings.TrimSpace(list[i])
		i, err := strconv.ParseInt(s, 10, 0)
		if err != nil {
			return err
		}
		*l = append(*l, int(i))
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
