// Package gli is a CLI parsing and mapping library.
//
// type globalCmd struct {
//     Name string
//     Age  int
// }
// func (g *globalCmd) Run() error {
//     // :
// }
// app := gli.New(&globalCmd{})
// err := app.Run(os.Args)
package gli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"bitbucket.org/shu_go/cliparser"
)

var (
	ErrNotDefined     = fmt.Errorf("not defined")
	ErrNotRunnable    = fmt.Errorf("command not runnable")
	ErrOptCanNotBeSet = fmt.Errorf("option can not be set")
)

type App struct {
	// tag keys
	CliTag, HelpTag, UsageTag, DefaultTag, EnvTag string

	// global help header
	Name, Desc, Usage, Version string
	// global help footer
	Copyright string

	SuppressErrorOutput bool
	Stdout, Stderr      *os.File

	parser cliparser.Parser
	root   *command
}

func New(ptrSt interface{}) App {
	v := reflect.ValueOf(ptrSt)
	if v.Kind() != reflect.Ptr && v.Elem().Kind() != reflect.Struct {
		panic("not a pointer to a struct")
	}

	app := App{
		parser: cliparser.New(),
		root: &command{
			SelfV: v,
		},

		CliTag:     "cli",
		HelpTag:    "help",
		UsageTag:   "usage",
		DefaultTag: "default",
		EnvTag:     "env",

		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	if err := app.scanMeta(v.Type(), app.root); err != nil {
		panic(err.Error())
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

func (g *App) Rescan(ptrSt interface{}) error {
	v := reflect.ValueOf(ptrSt)
	if v.Kind() != reflect.Ptr && v.Elem().Kind() != reflect.Struct {
		panic("not a pointer to a struct")
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

		name := ft.Name
		tag := ft.Tag

		names := []string{}
		var env string
		var defvalue string
		var help string
		var usage string
		var placeholder string

		// names
		if tv, ok := tag.Lookup(g.CliTag); ok {
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
			for _, n := range names {
				g.parser.HintCommand(n)
			}

			err := g.scanMeta(ft.Type, sub)
			if err != nil {
				return err
			}
		} else {
			cmd.Options = append(cmd.Options, &option{
				Names:       names,
				Env:         env,
				DefValue:    defvalue,
				Help:        help,
				Placeholder: placeholder,
				fieldIdx:    i,
			})

			//HINT
			for _, n := range names {
				if len(n) > 1 {
					g.parser.HintLongName(n)
				}
				if !isbool {
					g.parser.HintWithArg(n)
				}
			}
		}
	}

	return nil
}

// AddExtraCommand adds a sub command.
func (g *App) AddExtraCommand(ptrSt interface{}, names, help string, inits ...extraCmdInit) {
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
	}
	cmd := command{
		Names:  nameslice,
		Help:   help,
		SelfV:  v,
		Parent: g.root,
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

func (g *App) Parse(args []string) (tgt interface{}, tgtargs []string, appRunErr error) {
	return g.exec(args, false)
}

func (g *App) Run(args []string) (appRunErr error) {
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

	g.parser.Feed(args)
	if err := g.parser.Parse(); err != nil {
		return nil, nil, err
	}

	for {
		c := g.parser.GetComponent()
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
			o := cmd.FindOptionExact(c.Name)
			if o == nil {
				if !g.SuppressErrorOutput {
					fmt.Fprintf(g.Stdout, "option %s %v\n\n", c.Name, ErrNotDefined)
					g.Help(g.Stdout)
				}
				return nil, nil, ErrNotDefined
			}

			err := setOptValue(o.OwnerV.Elem().Field(o.fieldIdx), c.Arg)
			if err != nil {
				if !g.SuppressErrorOutput {
					fmt.Fprintf(g.Stderr, "option %s: %v\n\n", c.Name, err)
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
					fmt.Fprintf(g.Stderr, "command %s %v\n\n", c.Name, ErrNotDefined)
					g.Help(g.Stdout)
				}
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
