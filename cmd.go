package gli

import (
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	runewidth "github.com/mattn/go-runewidth"
)

type cmd struct {
	names []string

	subs   []*cmd
	extras []*cmd

	opts []*opt
	args []string

	help  string
	usage string

	self     interface{}
	fieldIdx int // physical
	holder   reflect.Value
}

func (c cmd) Help(w io.Writer) {
	if len(c.names) > 0 {
		name := longestName(c.names)
		fmt.Fprintf(w, "command %s - %s\n", name, c.help)
		fmt.Fprintln(w)
	}

	if len(c.subs)+len(c.extras) > 0 {
		fmt.Fprintln(w, "sub commands:")

		var subs []*cmd
		subs = append(subs, c.subs...)
		subs = append(subs, c.extras...)

		var names []string
		var helps []string
		width := 0
		for _, s := range subs {
			snames := s.names
			sort.Slice(snames, func(i, j int) bool { return len(snames[i]) > len(snames[j]) })
			n := strings.Join(s.names, ", ")
			names = append(names, n)
			helps = append(helps, s.help)

			w := runewidth.StringWidth(n)
			if width < w {
				width = w
			}
		}

		width += ((width + 1) % 8)

		for i, n := range names {
			spaces := strings.Repeat(" ", width-len(n))
			fmt.Fprintf(w, " %s%s%s\n", n, spaces, helps[i])
		}
		fmt.Fprintln(w)
	}

	if len(c.opts) > 0 {
		fmt.Fprintln(w, "options:")

		var names []string
		var helps []string
		var defdesc []string
		width := 0

		for _, o := range c.opts {
			var onames []string
			onames = append(onames, o.names...)
			for i, n := range onames {
				if len(n) == 1 {
					onames[i] = "-" + n
				} else {
					onames[i] = "--" + n
				}
			}

			sort.Slice(onames, func(i, j int) bool { return len(onames[i]) < len(onames[j]) })
			n := strings.Join(onames, ", ")
			if o.placeholder != "" {
				n += " " + o.placeholder
			}
			names = append(names, n)
			helps = append(helps, o.help)
			defdesc = append(defdesc, o.defvalue)

			w := runewidth.StringWidth(n)
			if width < w {
				width = w
			}
		}

		width += ((width + 1) % 8)

		for i, n := range names {
			spaces := strings.Repeat(" ", 1+width-runewidth.StringWidth(n))

			def := ""
			if len(defdesc[i]) > 0 {
				def = " (default: " + defdesc[i] + ")"
			}

			fmt.Fprintf(w, " %s%s%s%s\n", n, spaces, helps[i], def)
		}
		fmt.Fprintln(w)
	}

	if len(c.usage) > 0 {
		fmt.Fprintf(w, "Usage:\n  %v\n", strings.Replace(strings.TrimSpace(c.usage), "\n", "\n  ", -1))
	}
}

func (c cmd) String() string {
	opts := []string{}
	for _, o := range c.opts {
		opts = append(opts, o.String())
	}

	subs := []string{}
	for _, s := range c.extras {
		subs = append(subs, s.String())
	}
	for _, s := range c.subs {
		subs = append(subs, s.String())
	}

	return fmt.Sprintf("cmd{names=%v, help=%v opts=%v, subs=%v}", c.names, c.help, opts, subs)
}

func (c cmd) findOpt(name string) *opt {
	for _, o := range c.opts {
		for _, n := range o.names {
			if n == name {
				return o
			}
		}
	}
	return nil
}

func (c cmd) findSubCmd(name string) *cmd {
	for _, s := range c.extras {
		for _, n := range s.names {
			if n == name {
				return s
			}
		}
	}
	for _, s := range c.subs {
		for _, n := range s.names {
			if n == name {
				return s
			}
		}
	}
	return nil
}

func longestName(names []string) string {
	name := ""
	for _, n := range names {
		if len(name) < len(n) {
			name = n
		}
	}
	return name
}
