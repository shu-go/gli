package gli

import (
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"bitbucket.org/shu/rog"
)

type App struct {
	cmd cmd

	// global help header
	Name, Desc, Usage, Version string
	// global help footer
	Copyright string
}

func New(ptrSt interface{}) App {
	vroot := reflect.ValueOf(ptrSt)
	if vroot.Kind() != reflect.Ptr && vroot.Elem().Kind() != reflect.Struct {
		panic("not a pointer to a struct")
	}

	app := App{
		cmd: cmd{self: ptrSt},
	}

	err := gather(vroot, &app.cmd)
	if err != nil {
		panic(err.Error())
	}

	return app
}

func (app App) Help(w io.Writer) {
	appinfo := app.Name
	if app.Desc != "" {
		appinfo += " - " + app.Desc
	}
	if app.Version != "" {
		appinfo += "(" + app.Version + ")"
	}
	if appinfo != "" {
		fmt.Fprintf(w, "%s\n\n", appinfo)
	}

	app.cmd.usage = app.Usage

	app.cmd.Help(w)

	fmt.Fprintln(w, `
Help sub commands:
  help          `+app.Name+` help subcommnad subsubcommand
  version       show version`)

	if app.Copyright != "" {
		fmt.Fprintf(w, "\n%s\n", app.Copyright)
	}
}

func (app *App) AddExtraCommand(ptrSt interface{}, names, help string) {
	vroot := reflect.ValueOf(ptrSt)
	if vroot.Kind() != reflect.Ptr && vroot.Elem().Kind() != reflect.Struct {
		panic("not a pointer to a struct")
	}
	if len(names) == 0 {
		panic("name the extra command")
	}

	nameslice := strings.Split(names, ",")
	for i, n := range nameslice {
		nameslice[i] = strings.TrimSpace(n)
	}
	c := cmd{names: nameslice, help: help, self: ptrSt}

	err := gather(vroot, &c)
	if err != nil {
		panic(err.Error())
	}

	app.cmd.extras = append(app.cmd.extras, &c)

}

func (app App) Run(args []string) (appRunErr error) {
	c := &app.cmd

	setDefaultValues(c)

	if len(args) == len(os.Args) && len(args) > 0 && args[0] == os.Args[0] {
		args = args[1:]
	}

	cmdStack := []*cmd{c}

	rog.Debug("call Init root")
	_, defErr := call("Init", c.self, cmdStack, c.args)
	if defErr != nil {
		return defErr
	}

	var name string
	var o *opt

	helpMode := false

	arg := strings.TrimSpace(strings.Join(args, " "))
	for {
		t, l := token(arg)
		if l == 0 {
			break
		}
		arg = arg[l:]

		if len(cmdStack) == 1 && (t == "help" || t == "--help" || t == "-h") {
			helpMode = true
			continue
		}

		if len(cmdStack) == 1 && (t == "version") {
			fmt.Fprintln(os.Stdout, app.Version)
			return nil
		}

		if name == "" && strings.HasPrefix(t, "-") {
			longOpt := strings.HasPrefix(t, "--")

			name = strings.TrimLeft(t, "-")
			o = c.findOpt(name)
			if o == nil && !longOpt {
				for i, ch := range name {
					o = c.findOpt(string(ch))
					if o == nil {
						fmt.Fprintf(os.Stdout, "option %s %v\n\n", string(ch), ErrNotDefined)
						app.Help(os.Stdout)
						return ErrNotDefined
					}
					if i < len(name)-1 {
						o.holder.Field(o.fieldIdx).Set(reflect.ValueOf(true))
					}
				}
			}

			if o == nil {
				fmt.Fprintf(os.Stderr, "option %s %v\n\n", name, ErrNotDefined)
				app.Help(os.Stdout)
				return ErrNotDefined
			}

			if o != nil {
				if o.holder.Field(o.fieldIdx).Kind() == reflect.Bool {
					o.holder.Field(o.fieldIdx).Set(reflect.ValueOf(true))
					name = ""
					o = nil
				}
			}

		} else if t == "=" {
			//nop

		} else if name != "" && o != nil {
			//o.holder.Field(o.fieldIdx).Set(reflect.ValueOf(t))
			setOptValue(o.holder.Field(o.fieldIdx), t)

			name = ""

		} else if len(c.args) == 0 {
			sub := c.findSubCmd(t)
			if sub == nil {
				c.args = append(c.args, t)
			} else {
				c = sub
				cmdStack = append(cmdStack, c)

				rog.Debug("call Init ", c.names)
				_, defErr := call("Init", c.self, cmdStack, c.args)
				if defErr != nil {
					return defErr
				}
			}

		} else {
			c.args = append(c.args, t)
		}
	}

	if helpMode {
		funcName := "Help"

		rog.Debug("call "+funcName+" ", c.names)
		callErr, helpErr := call(funcName, c.self, cmdStack, c.args)
		if callErr == ErrNotRunnable {
			rog.Debug("call " + funcName + " root")
			callErr, helpErr = call(funcName, app.cmd.self, cmdStack, app.cmd.args)
		}

		if callErr != nil {
			if c.self == app.cmd.self {
				app.Help(os.Stdout)
			} else {
				c.Help(os.Stdout)
			}
		}

		return helpErr
	}

	// call Before->Run->After

	// Before/After
	// Before: root->sub->subsub
	// After: subsub->sub->root *deferred*

	for _, c := range cmdStack {
		rog.Debug("call Before ", c.names)
		callErr, beforeErr := call("Before", c.self, cmdStack, c.args)
		if callErr == nil && beforeErr != nil {
			fmt.Fprintf(os.Stderr, "%v\n", beforeErr)
			c.Help(os.Stdout)
			return beforeErr
		}

		defer func(c *cmd) {
			// After()
			rog.Debug("call After ", c.names)
			callErr, afterErr := call("After", c.self, cmdStack, c.args)
			if callErr != nil && appRunErr == nil {
				appRunErr = afterErr
			}
		}(c)
	}

	funcName := "Run"

	rog.Debug("call "+funcName+" ", c.names)
	callErr, runErr := call(funcName, c.self, cmdStack, c.args)

	if callErr != nil {
		if c == &app.cmd {
			app.Help(os.Stdout)
		} else {
			c.Help(os.Stdout)
		}
		return nil
	}

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "%v\n", runErr)
		return runErr
	}

	return nil
}

func gather(vtgt reflect.Value, tgt *cmd) error {
	vtgt = reflect.Indirect(vtgt)
	if vtgt.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < vtgt.NumField(); i++ {
		f := vtgt.Field(i)
		ft := vtgt.Type().Field(i)

		switch ft.Type.Kind() {
		case reflect.Map:
			panic("not supported yet")
		}

		iscmd := false
		{ // struct is skipped if a non-Parsable
			if f.Type().Kind() == reflect.Struct {
				_, ok := f.Interface().(Parsable)
				if !ok {
					_, ok := f.Addr().Interface().(Parsable)
					if !ok {
						iscmd = true
					}
				}
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
		if tv, ok := tag.Lookup("cli"); ok {
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
		// default value
		if tv, ok := tag.Lookup("default"); ok {
			defvalue = strings.TrimSpace(tv)
		}
		// default environment variable
		if tv, ok := tag.Lookup("env"); ok {
			env = strings.TrimSpace(tv)
		}
		// help description
		if tv, ok := tag.Lookup("help"); ok {
			help = strings.TrimSpace(tv)
		}
		// usage description
		if tv, ok := tag.Lookup("usage"); ok {
			usage = strings.TrimSpace(tv)
		}

		if iscmd /* f.Kind() == reflect.Struct */ {
			sub := &cmd{
				names:    names,
				help:     help,
				usage:    usage,
				self:     f.Addr().Interface(),
				fieldIdx: i,
				holder:   vtgt,
			}
			tgt.subs = append(tgt.subs, sub)

			err := gather(f, sub)
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
				holder:      vtgt,
			})
		}
	}

	return nil
}

func call(funcName string, cmd interface{}, cmdStack []*cmd, args []string) (callErr, userErr error) {
	if cmd == nil {
		return ErrNotRunnable, nil
	}
	methv := reflect.ValueOf(cmd).MethodByName(funcName)
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

	var retv []reflect.Value
	retv = methv.Call(argv)

	if len(retv) == 0 {
		return nil, nil
	}

	mayerr := retv[len(retv)-1].Interface()
	if err, ok := mayerr.(error); ok {
		return nil, err
	}
	return nil, nil
}

func findStructByType(stack []*cmd, typ reflect.Type) interface{} {
	for i := len(stack) - 1; i >= 0; i-- {
		e := stack[i].self
		et := reflect.TypeOf(e)
		if et == typ || et.Kind() == reflect.Ptr && et.Elem() == typ {
			return e
		}
	}
	return nil
}

func token(src string) (string, int) {
	src = strings.TrimSpace(src)
	if len(src) == 0 {
		return "", 0
	}

	switch src[0] {
	case '-':
		for i := 0; i < len(src); i++ {
			if src[i] == '=' {
				return src[:i], i + 1
			}
			if src[i] == '"' {
				return src[:i], i + 1
			}
			if src[i] == ' ' {
				return src[:i], i + 1
			}
		}
		return src, len(src)

	case '=':
		return "=", 1

	case '"':
		for i := 0; i < len(src); i++ {
			if src[i] == '\\' {
				i++
				continue
			}
			if src[i] == '"' {
				return src[:i], i + 1
			}
		}
		return src, len(src)

	default:
		for i := 0; i < len(src); i++ {
			if src[i] == '\\' {
				i++
				continue
			}
			if src[i] == '=' {
				return src[:i], i + 1
			}
			if src[i] == '"' {
				return src[:i], i + 1
			}
			if src[i] == ' ' {
				return src[:i], i + 1
			}
		}
		return src, len(src)
	}

	return "", 0
}
