package gli

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	runewidth "github.com/mattn/go-runewidth"
)

type command struct {
	names []string

	parent *command
	subs   []*command
	extras []*command

	options []*option

	args []string

	help  string
	usage string

	selfV    reflect.Value
	ownerV   reflect.Value
	fieldIdx int

	autoNoBoolOptions bool
}

func (c command) longestName() string {
	maxlen := -1
	var maxname string
	for _, n := range c.names {
		nlen := len(n)
		if nlen > maxlen {
			maxlen = nlen
			maxname = n
		}
	}

	return maxname
}

func (c command) longestNameStack() []string {
	var s []string

	for cmd := &c; cmd != nil; cmd = cmd.parent {
		n := cmd.longestName()
		if n == "" {
			break
		}
		s = append(s, n)
	}

	// reverse
	for i := 0; i < len(s)/2; i++ {
		//swap
		s[i], s[len(s)-i-1] = s[len(s)-i-1], s[i]
	}

	return s
}

func (c *command) findOptionExact(name string) *option {
	for _, o := range c.options {
		for _, n := range o.names {
			if n == name {
				return o
			}
		}
	}
	return nil
}

func (c *command) findCommandExact(name string) (cmd *command, isextra bool) {
	for _, c := range c.subs {
		for _, n := range c.names {
			if n == name {
				return c, false
			}
		}
	}
	for _, c := range c.extras {
		for _, n := range c.names {
			if n == name {
				return c, true
			}
		}
	}
	return nil, false
}

func (c *command) setMembersReferMe() {
	for _, o := range c.options {
		o.ownerV = c.selfV
	}
	for _, s := range c.subs {
		s.ownerV = c.selfV
	}
}

func (c *command) setDefaultValues() {
	for _, o := range c.options {
		if o.defValue != "" {
			var dummy bool
			_ = setOptValue(o.ownerV.Elem().Field(o.fieldIdx), o.defValue, true, &dummy)
			o.assigned = true
		}
		if o.env != "" {
			envvalue := os.Getenv(o.env)
			if envvalue != "" {
				var dummy bool
				_ = setOptValue(o.ownerV.Elem().Field(o.fieldIdx), envvalue, true, &dummy)
				o.assigned = true
			}
		}
	}
}

func (c command) outputHelp(w io.Writer) {
	if len(c.names) > 0 {
		name := longestName(c.names)
		fmt.Fprintf(w, "command %s - %s\n", name, c.help)
	}

	if len(c.subs)+len(c.extras) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Sub commands:")

		var subs []*command
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

		width += 2

		for i, n := range names {
			spaces := strings.Repeat(" ", width-len(n))
			fmt.Fprintf(w, "  %s%s%s\n", n, spaces, helps[i])
		}
	}

	if len(c.options) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Options:")

		var names []string
		var nonames []string
		var helps []string
		var defdesc []string
		var envs []string
		width := 0

		for _, o := range c.options {

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
			if c.autoNoBoolOptions && o.ownerV.Elem().Field(o.fieldIdx).Type().Kind() == reflect.Bool {
				b, err := strconv.ParseBool(o.defValue)
				if err != nil {
					b = false
				}
				if b {
					oname := strings.TrimLeft(onames[len(onames)-1], "-")
					nonames = append(nonames, "--no-"+oname)
				} else {
					nonames = append(nonames, "")
				}
			} else {
				nonames = append(nonames, "")
			}

			helps = append(helps, o.help)
			if o.defDesc != "" {
				defdesc = append(defdesc, o.defDesc)
			} else {
				defdesc = append(defdesc, o.defValue)
			}
			envs = append(envs, o.env)

			w := runewidth.StringWidth(n)
			if width < w {
				width = w
			}
		}

		width += 2

		for i, n := range names {
			spaces := strings.Repeat(" ", width-runewidth.StringWidth(n))

			var des []string
			if len(defdesc[i]) > 0 {
				des = append(des, "default: "+defdesc[i])
			}
			if len(envs[i]) > 0 {
				des = append(des, "env: "+envs[i])
			}
			var de string
			if len(des) > 0 {
				de = " (" + strings.Join(des, " ") + ")"
			}

			fmt.Fprintf(w, "  %s%s%s%s\n", n, spaces, helps[i], de)

			if nonames[i] != "" {
				fmt.Fprintf(w, "    %s\n", nonames[i])
			}
		}
	}

	curr := &c
	for {
		curr = curr.parent
		if curr == nil {
			break
		}

		if len(curr.options) > 0 {
			currname := "Global"
			if len(curr.names) > 0 {
				currname = "Outer " + curr.names[0]
			}

			fmt.Fprintln(w)
			fmt.Fprintf(w, "%s Options:\n", currname)

			var names []string
			var nonames []string
			var helps []string
			var defdesc []string
			var envs []string
			width := 0

			for _, o := range curr.options {
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
				if curr.autoNoBoolOptions && o.ownerV.Elem().Field(o.fieldIdx).Type().Kind() == reflect.Bool {
					b, err := strconv.ParseBool(o.defValue)
					if err != nil {
						b = false
					}
					if b {
						oname := strings.TrimLeft(onames[len(onames)-1], "-")
						nonames = append(nonames, "--no-"+oname)
					} else {
						nonames = append(nonames, "")
					}
				} else {
					nonames = append(nonames, "")
				}

				helps = append(helps, o.help)
				if o.defDesc != "" {
					defdesc = append(defdesc, o.defDesc)
				} else {
					defdesc = append(defdesc, o.defValue)
				}
				envs = append(envs, o.env)

				w := runewidth.StringWidth(n)
				if width < w {
					width = w
				}
			}

			width += 2

			for i, n := range names {
				spaces := strings.Repeat(" ", width-runewidth.StringWidth(n))

				var des []string
				if len(defdesc[i]) > 0 {
					des = append(des, "default: "+defdesc[i])
				}
				if len(envs[i]) > 0 {
					des = append(des, "env: "+envs[i])
				}
				var de string
				if len(des) > 0 {
					de = " (" + strings.Join(des, " ") + ")"
				}

				fmt.Fprintf(w, "  %s%s%s%s\n", n, spaces, helps[i], de)

				if nonames[i] != "" {
					fmt.Fprintf(w, "    %s\n", nonames[i])
				}
			}
		}
	}

	if len(c.usage) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Usage:\n  %v\n", strings.Replace(strings.TrimSpace(c.usage), "\n", "\n  ", -1))
	}
}

type extraCmdInit func(*command)

// Usage is an optional argument to AddExtracommand.
func Usage(usage string) extraCmdInit {
	return func(c *command) {
		c.usage = usage
	}
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
