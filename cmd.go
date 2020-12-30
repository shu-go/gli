package gli

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strings"

	runewidth "github.com/mattn/go-runewidth"
)

type command struct {
	Names []string

	Parent *command
	Subs   []*command
	Extras []*command

	Options []*option

	Args []string

	Help  string
	Usage string

	SelfV    reflect.Value
	OwnerV   reflect.Value
	fieldIdx int
}

func (c command) String() string {
	return fmt.Sprintf("command{Names:%v, Opts:%v, Subs:%v, Extras:%v, Args:%v}", c.Names, c.Options, c.Subs, c.Extras, c.Args)
}

func (c command) LongestName() string {
	maxlen := -1
	var maxname string
	for _, n := range c.Names {
		nlen := len(n)
		if nlen > maxlen {
			maxlen = nlen
			maxname = n
		}
	}

	return maxname
}

func (c command) LongestNameStack() []string {
	var s []string

	for cmd := &c; cmd != nil; cmd = cmd.Parent {
		n := cmd.LongestName()
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

func (c *command) FindOptionExact(name string) *option {
	for _, o := range c.Options {
		for _, n := range o.Names {
			if n == name {
				return o
			}
		}
	}
	return nil
}

func (c *command) FindCommandExact(name string) (cmd *command, isextra bool) {
	for _, c := range c.Subs {
		for _, n := range c.Names {
			if n == name {
				return c, false
			}
		}
	}
	for _, c := range c.Extras {
		for _, n := range c.Names {
			if n == name {
				return c, true
			}
		}
	}
	return nil, false
}

func (c *command) setMembersReferMe() {
	for _, o := range c.Options {
		o.OwnerV = c.SelfV
	}
	for _, s := range c.Subs {
		s.OwnerV = c.SelfV
	}
}

func (c *command) setDefaultValues() {
	for _, o := range c.Options {
		if o.DefValue != "" {
			var dummy bool
			_ = setOptValue(o.OwnerV.Elem().Field(o.fieldIdx), o.DefValue, true, &dummy)
		}
		if o.Env != "" {
			envvalue := os.Getenv(o.Env)
			if envvalue != "" {
				var dummy bool
				_ = setOptValue(o.OwnerV.Elem().Field(o.fieldIdx), envvalue, true, &dummy)
			}
		}
	}
}

func (c command) OutputHelp(w io.Writer) {
	if len(c.Names) > 0 {
		name := longestName(c.Names)
		fmt.Fprintf(w, "command %s - %s\n", name, c.Help)
	}

	if len(c.Subs)+len(c.Extras) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Sub commands:")

		var subs []*command
		subs = append(subs, c.Subs...)
		subs = append(subs, c.Extras...)

		var names []string
		var helps []string
		width := 0
		for _, s := range subs {
			snames := s.Names
			sort.Slice(snames, func(i, j int) bool { return len(snames[i]) > len(snames[j]) })
			n := strings.Join(s.Names, ", ")
			names = append(names, n)
			helps = append(helps, s.Help)

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

	if len(c.Options) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Options:")

		var names []string
		var helps []string
		var defdesc []string
		width := 0

		for _, o := range c.Options {
			var onames []string
			onames = append(onames, o.Names...)
			for i, n := range onames {
				if len(n) == 1 {
					onames[i] = "-" + n
				} else {
					onames[i] = "--" + n
				}
			}

			sort.Slice(onames, func(i, j int) bool { return len(onames[i]) < len(onames[j]) })
			n := strings.Join(onames, ", ")
			if o.Placeholder != "" {
				n += " " + o.Placeholder
			}
			names = append(names, n)
			helps = append(helps, o.Help)
			if o.DefDesc != "" {
				defdesc = append(defdesc, o.DefDesc)
			} else {
				defdesc = append(defdesc, o.DefValue)
			}

			w := runewidth.StringWidth(n)
			if width < w {
				width = w
			}
		}

		width += 2

		for i, n := range names {
			spaces := strings.Repeat(" ", width-runewidth.StringWidth(n))

			def := ""
			if len(defdesc[i]) > 0 {
				def = " (default: " + defdesc[i] + ")"
			}

			fmt.Fprintf(w, "  %s%s%s%s\n", n, spaces, helps[i], def)
		}
	}

	curr := &c
	for {
		curr = curr.Parent
		if curr == nil {
			break
		}

		if len(curr.Options) > 0 {
			currname := "Global"
			if len(curr.Names) > 0 {
				currname = "Outer " + curr.Names[0]
			}

			fmt.Fprintln(w)
			fmt.Fprintf(w, "%s Options:\n", currname)

			var names []string
			var helps []string
			var defdesc []string
			width := 0

			for _, o := range curr.Options {
				var onames []string
				onames = append(onames, o.Names...)
				for i, n := range onames {
					if len(n) == 1 {
						onames[i] = "-" + n
					} else {
						onames[i] = "--" + n
					}
				}

				sort.Slice(onames, func(i, j int) bool { return len(onames[i]) < len(onames[j]) })
				n := strings.Join(onames, ", ")
				if o.Placeholder != "" {
					n += " " + o.Placeholder
				}
				names = append(names, n)
				helps = append(helps, o.Help)
				if o.DefDesc != "" {
					defdesc = append(defdesc, o.DefDesc)
				} else {
					defdesc = append(defdesc, o.DefValue)
				}

				w := runewidth.StringWidth(n)
				if width < w {
					width = w
				}
			}

			width += 2

			for i, n := range names {
				spaces := strings.Repeat(" ", width-runewidth.StringWidth(n))

				def := ""
				if len(defdesc[i]) > 0 {
					def = " (default: " + defdesc[i] + ")"
				}

				fmt.Fprintf(w, "  %s%s%s%s\n", n, spaces, helps[i], def)
			}
		}
	}

	if len(c.Usage) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Usage:\n  %v\n", strings.Replace(strings.TrimSpace(c.Usage), "\n", "\n  ", -1))
	}
}

type extraCmdInit func(*command)

// Usage is an optional argument to AddExtracommand.
func Usage(usage string) extraCmdInit {
	return func(c *command) {
		c.Usage = usage
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
