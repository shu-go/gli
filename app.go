package gli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// App is a parsed global command holder.
type App struct {
	cmd cmd

	CliTag, HelpTag, UsageTag, DefaultTag, EnvTag string

	SuppressErrorOutput bool
	Stdout, Stderr      *os.File

	// global help header
	Name, Desc, Usage, Version string
	// global help footer
	Copyright string
}

// New creates new App object.
func New(ptrSt interface{}) App {
	v := reflect.ValueOf(ptrSt)
	if v.Kind() != reflect.Ptr && v.Elem().Kind() != reflect.Struct {
		panic("not a pointer to a struct")
	}

	app := App{
		cmd: cmd{v: v},

		CliTag:     "cli",
		HelpTag:    "help",
		UsageTag:   "usage",
		DefaultTag: "default",
		EnvTag:     "env",

		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	err := app.gather(v.Type(), &app.cmd)
	if err != nil {
		panic(err.Error())
	}

	if exe, err := os.Executable(); err == nil {
		app.Name = filepath.Base(exe)
	}

	return app
}

func (app *App) Rescan(ptrSt interface{}) error {
	v := reflect.ValueOf(ptrSt)
	if v.Kind() != reflect.Ptr && v.Elem().Kind() != reflect.Struct {
		panic("not a pointer to a struct")
	}

	app.cmd = cmd{v: v}

	return app.gather(v.Type(), &app.cmd)
}

// Help displays help messages.
func (app App) Help(w io.Writer) {
	appinfo := app.Name
	if app.Desc != "" {
		appinfo += " - " + app.Desc
	}
	if app.Version != "" {
		appinfo += "(" + app.Version + ")"
	}
	if appinfo != "" {
		fmt.Fprintf(w, "%s\n", appinfo)
	}

	app.cmd.usage = app.Usage

	app.cmd.Help(w)

	fmt.Fprintln(w, `
Help sub commands:
  help     `+app.Name+` help subcommnad subsubcommand
  version  show version`)

	if app.Copyright != "" {
		fmt.Fprintf(w, "\n%s\n", app.Copyright)
	}
}

// AddExtraCommand adds a sub command.
func (app *App) AddExtraCommand(ptrSt interface{}, names, help string, inits ...extraCmdInit) {
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
	}
	c := cmd{
		names:  nameslice,
		help:   help,
		v:      v,
		parent: &app.cmd,
	}

	for _, init := range inits {
		init(&c)
	}

	err := app.gather(v.Type(), &c)
	if err != nil {
		panic(err.Error())
	}

	app.cmd.extras = append(app.cmd.extras, &c)
}

// Run parses args and fills global command struct that is passed via
// New(&globalCmd).
func (app App) Run(args []string) (appRunErr error) {
	_, _, err := app.exec(args, true)
	return err
}

// Parse parses args and fills global command struct that is passed via
// New(&globalCmd) and returns it.
// The Before/Run/After method of a command is not called. (Init method is colled)
func (app App) Parse(args []string) (tgt interface{}, tgtargs []string, appRunErr error) {
	return app.exec(args, false)
}

func (app App) exec(args []string, doRun bool) (tgt interface{}, tgtargs []string, appRunErr error) {
	c := &app.cmd

	if len(args) == len(os.Args) && len(args) > 0 && args[0] == os.Args[0] {
		args = make([]string, len(os.Args)-1)
		copy(args, os.Args[1:])
	}

	cmdStack := []*cmd{c}
	c.setMembersReferMe()
	c.setDefaultValues()

	_, defErr := call("Init", c.v, cmdStack, c.args)
	if defErr != nil {
		return nil, nil, defErr
	}

	var name string
	var o *opt

	helpMode := false

	for {
		t, l := token(&args)
		if l == 0 {
			break
		}

		if len(cmdStack) == 1 && (t == "help" || t == "--help" || t == "-h") {
			helpMode = true
			continue
		}

		if len(cmdStack) == 1 && (t == "version") {
			fmt.Fprintln(app.Stdout, app.Version)
			return nil, nil, nil
		}

		if name == "" && strings.HasPrefix(t, "-") {
			longOpt := strings.HasPrefix(t, "--")

			name = strings.TrimLeft(t, "-")
			o = c.findOpt(name)
			if o == nil && !longOpt {
				for i, ch := range name {
					o = c.findOpt(string(ch))
					if o == nil {
						if !app.SuppressErrorOutput {
							fmt.Fprintf(app.Stdout, "option %s %v\n\n", string(ch), ErrNotDefined)
							app.Help(app.Stdout)
						}
						return nil, nil, ErrNotDefined
					}

					o.pv = c.v
					if i < len(name)-1 {
						o.pv.Elem().Field(o.fieldIdx).Set(reflect.ValueOf(true))
					}
				}
			}

			if o == nil {
				if !app.SuppressErrorOutput {
					fmt.Fprintf(app.Stderr, "option %s %v\n\n", name, ErrNotDefined)
					app.Help(app.Stdout)
				}
				return nil, nil, ErrNotDefined
			}

			o.pv = c.v

			if o.pv.Elem().Field(o.fieldIdx).Kind() == reflect.Bool {
				o.pv.Elem().Field(o.fieldIdx).Set(reflect.ValueOf(true))
				name = ""
				o = nil
			}

		} else if t == "=" {
			//nop

		} else if name != "" && o != nil {
			//reflect.ValueOf(o.cmd.self).Field(o.fieldIdx).Set(reflect.ValueOf(t))
			err := setOptValue(o.pv.Elem().Field(o.fieldIdx), t)
			if err != nil {
				if !app.SuppressErrorOutput {
					fmt.Fprintf(app.Stderr, "option %s: %v\n\n", name, err)
					app.Help(app.Stdout)
				}
				return nil, nil, err
			}

			name = ""

		} else if len(c.args) == 0 {
			if len(c.subs)+len(c.extras) > 0 {
				sub, extra := c.findSubCmd(t)
				if sub == nil {
					if !app.SuppressErrorOutput {
						fmt.Fprintf(app.Stderr, "command %s %v\n\n", t, ErrNotDefined)
						app.Help(app.Stdout)
					}
					return nil, nil, ErrNotDefined
				}

				if !extra {
					sub.pv = c.v
					subt := c.v.Type().Elem().Field(sub.fieldIdx).Type
					if subt.Kind() == reflect.Ptr {
						sub.v = reflect.New(subt.Elem())
						c.v.Elem().Field(sub.fieldIdx).Set(sub.v)
					} else {
						sub.v = c.v.Elem().Field(sub.fieldIdx).Addr()
					}
				}
				c = sub
				cmdStack = append(cmdStack, c)
				c.setMembersReferMe()
				c.setDefaultValues()

				_, defErr := call("Init", c.v, cmdStack, c.args)
				if defErr != nil {
					return nil, nil, defErr
				}
			} else {
				c.args = append(c.args, t)
			}

		} else {
			c.args = append(c.args, t)
		}
	}

	if helpMode {
		funcName := "Help"

		callErr, helpErr := call(funcName, c.v, cmdStack, c.args)
		if callErr == ErrNotRunnable {
			callErr, helpErr = call(funcName, app.cmd.v, cmdStack, app.cmd.args)
		}

		if callErr != nil {
			if c.v == app.cmd.v {
				app.Help(app.Stdout)
			} else {
				c.Help(app.Stdout)
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
			callErr, beforeErr := call("Before", c.v, cmdStack, c.args)
			if callErr == nil && beforeErr != nil {
				if !app.SuppressErrorOutput {
					fmt.Fprintf(app.Stderr, "%v\n", beforeErr)
					c.Help(app.Stdout)
				}
				return nil, nil, beforeErr
			}

			defer func(c *cmd) {
				// After()
				callErr, afterErr := call("After", c.v, cmdStack, c.args)
				if callErr != nil && appRunErr == nil {
					appRunErr = afterErr
				}
			}(c)
		}
	}

	if doRun {
		funcName := "Run"

		callErr, runErr := call(funcName, c.v, cmdStack, c.args)

		if callErr != nil {
			if c == &app.cmd {
				app.Help(app.Stdout)
			} else {
				c.Help(app.Stdout)
			}
			//return ErrNotDefined
			return nil, nil, nil
		}

		if runErr != nil {
			if !app.SuppressErrorOutput {
				fmt.Fprintf(app.Stderr, "%v\n", runErr)
			}
			return nil, nil, runErr
		}
	}

	return c.v.Interface(), c.args, nil
}

func (app App) gather(ttgt reflect.Type, tgt *cmd) error {
	if ttgt.Kind() == reflect.Ptr {
		ttgt = ttgt.Elem()
	}
	if ttgt.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < ttgt.NumField(); i++ {
		ft := ttgt.Field(i)

		switch ft.Type.Kind() {
		case reflect.Map:
			panic("not supported yet")
		}

		// goto next if not public field
		if ft.Name[:1] == strings.ToLower(ft.Name[:1]) {
			if ft.Name == "help" || ft.Name == "_" {
				tag := ft.Tag

				// help description
				if tv, ok := tag.Lookup(app.HelpTag); ok && tgt.help == "" {
					tgt.help = strings.TrimSpace(tv)
				}
				// usage description
				if tv, ok := tag.Lookup(app.UsageTag); ok && tgt.usage == "" {
					tgt.usage = strings.TrimSpace(tv)
				}
			}

			continue
		}

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
		if tv, ok := tag.Lookup(app.CliTag); ok {
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

		defvalue = strings.TrimSpace(tag.Get(app.DefaultTag))
		env = strings.TrimSpace(tag.Get(app.EnvTag))
		help = strings.TrimSpace(tag.Get(app.HelpTag))
		usage = strings.TrimSpace(tag.Get(app.UsageTag))

		if iscmd /* f.Kind() == reflect.Struct */ {
			sub := &cmd{
				names:    names,
				help:     help,
				usage:    usage,
				fieldIdx: i,
				parent:   tgt,
			}
			tgt.subs = append(tgt.subs, sub)

			err := app.gather(ft.Type, sub)
			if err != nil {
				return err
			}
		} else {
			tgt.opts = append(tgt.opts, &opt{
				names:       names,
				env:         env,
				defvalue:    defvalue,
				help:        help,
				placeholder: placeholder,
				fieldIdx:    i,
			})
		}
	}

	return nil
}

func call(funcName string, cmd reflect.Value, cmdStack []*cmd, args []string) (callErr, userErr error) {
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

func findStructByType(stack []*cmd, typ reflect.Type) interface{} {
	for i := len(stack) - 1; i >= 0; i-- {
		e := stack[i].v
		ei := stack[i].v.Interface()
		et := e.Type()
		if et == typ || et.Kind() == reflect.Ptr && et.Elem() == typ {
			return ei
		}
	}
	return nil
}

func token(args *[]string) (t string, length int) {
	if len(*args) == 0 {
		return "", 0
	}

	src := (*args)[0]
	if len(src) == 0 {
		return "", 0
	}

	switch src[0] {
	case '=':
		t, length = "=", 1

	default:
		for i := 0; i < len(src); i++ {
			if src[i] == '=' {
				t, length = src[:i], i //+ 1
				break
			}
		}
		if length == 0 { // centinel
			t, length = src, len(src)
		}
	}

	// consume curr token on args
	(*args)[0] = (*args)[0][length:]
	if len((*args)[0]) == 0 {
		*args = (*args)[1:]
	}
	return t, length
}

func isStructImplements(st reflect.Type, iface reflect.Type) bool {
	if st.Kind() != reflect.Struct && !(st.Kind() == reflect.Ptr && st.Elem().Kind() == reflect.Struct) {
		return false
	}

	return reflect.PtrTo(st).Implements(iface) || st.Implements(iface)
}
