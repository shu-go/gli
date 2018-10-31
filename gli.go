// Package gli is a CLI parsing and mapping library.
//
//     type globalCmd struct {
//         Name string
//         Age  int
//     }
//     func (g *globalCmd) Run() error {
//         // :
//     }
//     app := gli.NewWith(&globalCmd{})
//     err := app.Run(os.Args)
package gli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"bitbucket.org/shu_go/cliparser"
)

var (
	// ErrNotDefined means "an option or a subcommand is not defined in the passed struct".
	ErrNotDefined = fmt.Errorf("not defined")
	// ErrNotRunnable means "Run method is not defined for the passed struct".
	ErrNotRunnable = fmt.Errorf("command not runnable")
	// ErrOptCanNotBeSet is a reflect related error.
	ErrOptCanNotBeSet = fmt.Errorf("option can not be set")
)

// App contains parsing and parsed data.
type App struct {
	// global help header

	// Name is the app name. default: the name of the executable file
	Name string
	// Desc is a description of the app
	// {{Name}} - {{Desc}}
	Desc string
	// Usage is a long(multiline) usage text
	Usage string
	// Version numbers
	Version string

	// global help footer

	// Copyright the app auther has
	Copyright string

	// tag keys

	// CliTag is a tag key. default: `cli`
	CliTag string
	// HelpTag is a tag key. default: `help`
	HelpTag string
	// UsageTag is a tag key. default: `usage`
	UsageTag string
	// DefaultTag is a tag key. default: `default`
	DefaultTag string
	// EnvTag is a tag key. default: `env`
	EnvTag string

	// MyCommandABC => false(default): "mycommandabc" , true: "my-command-abc"
	HyphenedCommandName bool
	// MyOptionABC => false(default): "myoptionabc" , true: "my-option-abc"
	HyphenedOptionName bool
	// OptionsGrouped(default: true) allows -abc may be treated as -a -b -c.
	OptionsGrouped bool

	// SuppressErrorOutput is an option to suppresses on cli parsing error.
	SuppressErrorOutput bool
	Stdout, Stderr      *os.File

	parser cliparser.Parser
	root   *command
}

func New() App {
	app := App{
		parser: cliparser.New(),

		CliTag:     "cli",
		HelpTag:    "help",
		UsageTag:   "usage",
		DefaultTag: "default",
		EnvTag:     "env",

		HyphenedCommandName: false,
		HyphenedOptionName:  false,
		OptionsGrouped:      true,
		Stdout:              os.Stdout,
		Stderr:              os.Stderr,
	}

	//HINT
	app.parser.HintCommand("help")
	app.parser.HintCommand("version")
	app.parser.HintLongName("help")
	app.parser.HintLongName("version")

	if exe, err := os.Executable(); err == nil {
		app.Name = filepath.Base(exe)
	}

	return app
}

// NewWith is  New().Bind(ptrSt)
func NewWith(ptrSt interface{}) App {
	app := New()

	if err := app.Bind(ptrSt); err != nil {
		panic(err)
	}

	return app
}

// Bind updates option/command names with ptrSt.
// Extra commands are cleared.
func (g *App) Bind(ptrSt interface{}) error {
	v := reflect.ValueOf(ptrSt)
	if v.Kind() != reflect.Ptr && v.Elem().Kind() != reflect.Struct {
		panic("not a pointer to a struct")
	}

	// hmmmm....
	if !g.OptionsGrouped {
		g.parser.HintNoOptionsGrouped()
	}

	g.root = &command{SelfV: v}

	return g.scanMeta(v.Type(), g.root)
}

func (g *App) scanMeta(t reflect.Type, cmd *command) error {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)

		// goto next if not public field
		if ft.Name[:1] == strings.ToLower(ft.Name[:1]) {
			if ft.Name == "help" || ft.Name == "_" {
				tag := ft.Tag

				// help description
				if tv, ok := tag.Lookup(g.HelpTag); ok && cmd.Help == "" {
					cmd.Help = strings.TrimSpace(tv)
				}
				// usage description
				if tv, ok := tag.Lookup(g.UsageTag); ok && cmd.Usage == "" {
					cmd.Usage = strings.TrimSpace(tv)
				}
			}

			continue
		}

		isbool := ft.Type.Kind() == reflect.Bool
		iscmd := false
		// struct is skipped if a non-Parsable
		if ft.Type.Kind() == reflect.Struct || (ft.Type.Kind() == reflect.Ptr && ft.Type.Elem().Kind() == reflect.Struct) {
			if !isStructImplements(ft.Type, reflect.TypeOf((*Parsable)(nil)).Elem()) {
				iscmd = true
			}
		}

		name := g.arrangeName(ft.Name, iscmd)
		tag := ft.Tag

		names := []string{}
		var env string
		var defvalue string
		var help string
		var usage string
		var placeholder string

		// names
		if tv, ok := tag.Lookup(g.CliTag); ok {
			if tv == "-" || tv == "" {
				continue
			}

			clinames := strings.Split(tv, ",")
			for _, n := range clinames {
				n = strings.TrimSpace(n)
				if strings.Contains(n, "=") {
					nn := strings.Split(n, "=")
					n = strings.TrimSpace(nn[0])
					placeholder = strings.TrimSpace(nn[1])
				}
				names = append(names, n)
			}
		}
		if len(names) == 0 {
			names = append(names, strings.ToLower(name))
		}

		defvalue = strings.TrimSpace(tag.Get(g.DefaultTag))
		env = strings.TrimSpace(tag.Get(g.EnvTag))
		help = strings.TrimSpace(tag.Get(g.HelpTag))
		usage = strings.TrimSpace(tag.Get(g.UsageTag))

		if iscmd /* f.Kind() == reflect.Struct */ {
			sub := &command{
				Names:    names,
				Help:     help,
				Usage:    usage,
				fieldIdx: i,
				Parent:   cmd,
			}
			cmd.Subs = append(cmd.Subs, sub)

			//HINT
			lname := sub.LongestName()
			for _, n := range names {
				// a.out cmd1 cmd2
				// cmd2 [cmd1]
				g.parser.HintCommand(n, cmd.LongestNameStack())
				// a.out cmd1 help
				// help [cmd1]
				g.parser.HintCommand("help", cmd.LongestNameStack())
				// a.out cmd1 help cmd2
				// cmd2 [cmd1 help]
				g.parser.HintCommand(n, append(cmd.LongestNameStack(), "help"))
				//rog.Debug("HintAlias", n, lname)
				g.parser.HintAlias(n, lname)
			}

			err := g.scanMeta(ft.Type, sub)
			if err != nil {
				return err
			}
		} else {
			opt := &option{
				Names:       names,
				Env:         env,
				DefValue:    defvalue,
				Help:        help,
				Placeholder: placeholder,
				fieldIdx:    i,
			}
			cmd.Options = append(cmd.Options, opt)

			//HINT
			lname := opt.LongestName()
			for _, n := range names {
				if len(n) > 1 {
					g.parser.HintLongName(n, cmd.LongestNameStack())
				}
				if !isbool {
					g.parser.HintWithArg(n, cmd.LongestNameStack())
				}
				g.parser.HintAlias(n, lname)
			}
		}
	}

	return nil
}

// AddExtraCommand adds a sub command.
func (g *App) AddExtraCommand(ptrSt interface{}, names, help string, inits ...extraCmdInit) {
	if g.root == nil {
		panic("need Bind or use NewWith")
	}

	v := reflect.ValueOf(ptrSt)
	if v.Kind() != reflect.Ptr && v.Elem().Kind() != reflect.Struct {
		panic("not a pointer to a struct")
	}
	if len(names) == 0 {
		panic("name the extra command")
	}

	nameslice := strings.Split(names, ",")
	for i, n := range nameslice {
		nameslice[i] = strings.TrimSpace(n)
		g.parser.HintCommand(nameslice[i])
		g.parser.HintCommand(n, []string{"help"})
	}
	cmd := command{
		Names:  nameslice,
		Help:   help,
		SelfV:  v,
		Parent: g.root,
	}
	lname := cmd.LongestName()
	for _, n := range cmd.Names {
		g.parser.HintAlias(n, lname)
	}

	for _, init := range inits {
		init(&cmd)
	}

	err := g.scanMeta(v.Type(), &cmd)
	if err != nil {
		panic(err.Error())
	}

	g.root.Extras = append(g.root.Extras, &cmd)
}

// Parse parses args and returns results.
// tgt (interface{}) : a resultant struct
// tgtargs ([]string) : args of last subcommand
// err : parsing error
func (g *App) Parse(args []string) (tgt interface{}, tgtargs []string, err error) {
	if g.root == nil {
		panic("need Bind or use NewWith")
	}

	return g.exec(args, false)
}

// Run parses args and calls Run method of a subcommand.
func (g *App) Run(args []string) error {
	if g.root == nil {
		panic("need Bind or use NewWith")
	}

	_, _, err := g.exec(args, true)
	return err
}

func (g *App) exec(args []string, doRun bool) (tgt interface{}, tgtargs []string, appRunErr error) {
	cmd := g.root

	if len(args) == len(os.Args) && len(args) > 0 && args[0] == os.Args[0] {
		args = make([]string, len(os.Args)-1)
		copy(args, os.Args[1:])
	}

	cmdStack := []*command{cmd}
	cmd.setMembersReferMe()
	cmd.setDefaultValues()

	_, defErr := call("Init", cmd.SelfV, cmdStack, cmd.Args)
	if defErr != nil {
		return nil, nil, defErr
	}

	helpMode := false

	g.parser.Reset()
	g.parser.Feed(args)
	if err := g.parser.Parse(); err != nil {
		return nil, nil, err
	}

	for {
		c := g.parser.GetComponent()
		//rog.Debug("c", c)
		if c == nil {
			break
		}

		if c.Name == "help" {
			helpMode = true
			continue
		}

		if len(cmdStack) == 1 && (c.Name == "version") {
			fmt.Fprintln(g.Stdout, g.Version)
			return nil, nil, nil
		}

		switch c.Type {
		case cliparser.Arg:
			cmd.Args = append(cmd.Args, c.Arg)

		case cliparser.Option:
			//rog.Debug(c.Name)
			o := cmd.FindOptionExact(c.Name)
			if o == nil {
				if !g.SuppressErrorOutput {
					//rog.Debug("notdefined")
					fmt.Fprintf(g.Stdout, "option %q %v\n\n", c.Name, ErrNotDefined)

					var candidates []string
					for _, opt := range cmd.Options {
						for _, name := range opt.Names {
							if strings.HasPrefix(name, c.Name) {
								candidates = append(candidates, name)
								break
							} else if re, err := regexp.Compile("[" + name + "]"); err == nil {
								if len(re.ReplaceAllLiteralString(c.Name, "")) <= len(c.Name)/10 {
									candidates = append(candidates, name)
									break
								}
							}
						}
					}
					if len(candidates) > 0 {
						fmt.Fprintf(g.Stdout, "    maybe %v ?\n\n", candidates)
					}
					g.Help(g.Stdout)
				}
				//rog.Debug("notdefined")
				return nil, nil, ErrNotDefined
			}

			err := setOptValue(o.OwnerV.Elem().Field(o.fieldIdx), c.Arg)
			if err != nil {
				if !g.SuppressErrorOutput {
					fmt.Fprintf(g.Stderr, "option %q: %v\n\n", c.Name, err)
					g.Help(g.Stdout)
				}
				return nil, nil, err
			}

		case cliparser.Command: // may be an arg
			if len(cmd.Subs)+len(cmd.Extras) == 0 {
				cmd.Args = append(cmd.Args, c.Name) // command name? -> no, it's an arg
				continue
			}

			sub, isextra := cmd.FindCommandExact(c.Name)
			if sub == nil {
				if !g.SuppressErrorOutput {
					//rog.Debug("notdefined")
					fmt.Fprintf(g.Stderr, "command %q %v\n\n", c.Name, ErrNotDefined)
					g.Help(g.Stdout)
				}
				//rog.Debug("notdefined")
				return nil, nil, ErrNotDefined
			}

			if !isextra {
				sub.OwnerV = cmd.SelfV
				subt := cmd.SelfV.Type().Elem().Field(sub.fieldIdx).Type
				if subt.Kind() == reflect.Ptr {
					sub.SelfV = reflect.New(subt.Elem())
					cmd.SelfV.Elem().Field(sub.fieldIdx).Set(sub.SelfV)
				} else {
					sub.SelfV = cmd.SelfV.Elem().Field(sub.fieldIdx).Addr()
				}
			}
			cmd = sub
			cmdStack = append(cmdStack, cmd)
			cmd.setMembersReferMe()
			cmd.setDefaultValues()

			_, defErr := call("Init", cmd.SelfV, cmdStack, cmd.Args)
			if defErr != nil {
				return nil, nil, defErr
			}
		}
	}

	if helpMode {
		funcName := "Help"

		callErr, helpErr := call(funcName, cmd.SelfV, cmdStack, cmd.Args)
		if callErr == ErrNotRunnable {
			callErr, helpErr = call(funcName, g.root.SelfV, cmdStack, g.root.Args)
		}

		if callErr != nil {
			if cmd.SelfV == g.root.SelfV {
				g.Help(g.Stdout)
			} else {
				cmd.OutputHelp(g.Stdout)
			}
		}

		return nil, nil, helpErr
	}

	// call Before->Run->After

	// Before/After
	// Before: root->sub->subsub
	// After: subsub->sub->root *deferred*

	if doRun {
		for _, c := range cmdStack {
			callErr, beforeErr := call("Before", c.SelfV, cmdStack, c.Args)
			if callErr == nil && beforeErr != nil {
				if !g.SuppressErrorOutput {
					fmt.Fprintf(g.Stderr, "%v\n", beforeErr)
					c.OutputHelp(g.Stdout)
				}
				return nil, nil, beforeErr
			}

			defer func(cmd *command) {
				// After()
				callErr, afterErr := call("After", cmd.SelfV, cmdStack, cmd.Args)
				if callErr != nil && appRunErr == nil {
					appRunErr = afterErr
				}
			}(c)
		}
	}

	if doRun {
		funcName := "Run"

		callErr, runErr := call(funcName, cmd.SelfV, cmdStack, cmd.Args)

		if callErr != nil {
			if cmd == g.root {
				g.Help(g.Stdout)
			} else {
				cmd.OutputHelp(g.Stdout)
			}
			//return ErrNotDefined
			return nil, nil, nil
		}

		if runErr != nil {
			if !g.SuppressErrorOutput {
				fmt.Fprintf(g.Stderr, "%v\n", runErr)
			}
			return nil, nil, runErr
		}
	}

	return cmd.SelfV.Interface(), cmd.Args, nil
}

func (g *App) arrangeName(name string, iscmd bool) string {

	if iscmd && !g.HyphenedCommandName {
		return name
	}
	if !iscmd && !g.HyphenedOptionName {
		return name
	}

	result := make([]rune, 0, len(name))

	prevU := false
	for i, c := range name {
		if i != 0 && 'A' <= c && c <= 'Z' {
			if !prevU {
				result = append(result, '-')
			}
			prevU = true
		} else {
			prevU = false
		}

		result = append(result, c)
	}

	return string(result)
}

func isStructImplements(st reflect.Type, iface reflect.Type) bool {
	if st.Kind() != reflect.Struct && !(st.Kind() == reflect.Ptr && st.Elem().Kind() == reflect.Struct) {
		return false
	}

	return reflect.PtrTo(st).Implements(iface) || st.Implements(iface)
}

func call(funcName string, cmd reflect.Value, cmdStack []*command, args []string) (callErr, userErr error) {
	methv := cmd.MethodByName(funcName)
	if methv == (reflect.Value{}) {
		return ErrNotRunnable, nil
	}

	var argv []reflect.Value
	for i := 0; i < methv.Type().NumIn(); i++ {
		in := methv.Type().In(i)

		if in.Kind() == reflect.Struct {
			st := findStructByType(cmdStack, in)
			if st == nil {
				return ErrNotRunnable, nil
			}
			argv = append(argv, reflect.ValueOf(st).Elem())

		} else if in.Kind() == reflect.Ptr && in.Elem().Kind() == reflect.Struct {
			st := findStructByType(cmdStack, in)
			if st == nil {
				return ErrNotRunnable, nil
			}
			argv = append(argv, reflect.ValueOf(st))

		} else if in.Kind() == reflect.Slice && in.Elem().Kind() == reflect.String {
			// args
			argv = append(argv, reflect.ValueOf(args))

		} else {
			panic("*struct, struct or []string are allowed")
		}
	}

	retv := methv.Call(argv)

	return nil, returnErr(retv)
}

func returnErr(retv []reflect.Value) error {
	if len(retv) == 0 {
		return nil
	}

	mayerr := retv[len(retv)-1].Interface()
	if err, ok := mayerr.(error); ok {
		return err
	}

	return nil
}

func findStructByType(stack []*command, typ reflect.Type) interface{} {
	for i := len(stack) - 1; i >= 0; i-- {
		e := stack[i].SelfV
		ei := stack[i].SelfV.Interface()
		et := e.Type()
		if et == typ || et.Kind() == reflect.Ptr && et.Elem() == typ {
			return ei
		}
	}
	return nil
}

func setOptValue(opt reflect.Value, value string) error {
	if opt.Type().Kind() == reflect.Ptr {
		var pv reflect.Value
		if opt.IsNil() {
			pv = reflect.New(opt.Type().Elem())
		} else {
			pv = opt
		}

		err := setOptValue(pv.Elem(), value)
		if err != nil {
			return err
		}

		opt.Set(pv)
		return nil
	}
	p, ok := opt.Interface().(Parsable)
	if ok {
		return p.Parse(value)
	} else if opt.CanAddr() {
		p, ok := opt.Addr().Interface().(Parsable)
		if ok {
			return p.Parse(value)
		}
	}

	switch opt.Kind() {
	case reflect.String:
		opt.Set(reflect.ValueOf(value))
		return nil

	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		opt.Set(reflect.ValueOf(b))
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		size := int(opt.Type().Size())
		i, err := strconv.ParseInt(value, 10, size*8)
		if err != nil {
			return err
		}
		opt.Set(reflect.ValueOf(i).Convert(opt.Type()))
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		size := int(opt.Type().Size())
		i, err := strconv.ParseUint(value, 10, size*8)
		if err != nil {
			return err
		}
		opt.Set(reflect.ValueOf(i).Convert(opt.Type()))
		return nil

	case reflect.Float32, reflect.Float64:
		size := int(opt.Type().Size())
		f, err := strconv.ParseFloat(value, size*8)
		if err != nil {
			return err
		}
		opt.Set(reflect.ValueOf(f).Convert(opt.Type()))
		return nil
	}

	return ErrOptCanNotBeSet
}

// Help displays help messages.
func (g App) Help(w io.Writer) {
	if g.root == nil {
		panic("need Bind or use NewWith")
	}

	appinfo := g.Name
	if g.Desc != "" {
		appinfo += " - " + g.Desc
	}
	if g.Version != "" {
		appinfo += "(" + g.Version + ")"
	}
	if appinfo != "" {
		fmt.Fprintf(w, "%s\n", appinfo)
	}

	g.root.Usage = g.Usage

	g.root.OutputHelp(w)

	fmt.Fprintln(w, `
Help sub commands:
  help     `+g.Name+` help subcommnad subsubcommand
  version  show version`)

	if g.Copyright != "" {
		fmt.Fprintf(w, "\n%s\n", g.Copyright)
	}
}
