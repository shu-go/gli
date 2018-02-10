package gli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type App struct {
	cmd cmd

	SuppressErrorOutput bool

	// global help header
	Name, Desc, Usage, Version string
	// global help footer
	Copyright string
}

func New(ptrSt interface{}) App {
	v := reflect.ValueOf(ptrSt)
	if v.Kind() != reflect.Ptr && v.Elem().Kind() != reflect.Struct {
		panic("not a pointer to a struct")
	}

	app := App{
		cmd: cmd{v: v},
	}

	err := gather(v.Type(), &app.cmd)
	if err != nil {
		panic(err.Error())
	}

	if exe, err := os.Executable(); err == nil {
		app.Name = filepath.Base(exe)
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
		fmt.Fprintf(w, "%s\n", appinfo)
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
	c := cmd{names: nameslice, help: help, v: v}

	for _, init := range inits {
		init(&c)
	}

	err := gather(v.Type(), &c)
	if err != nil {
		panic(err.Error())
	}

	app.cmd.extras = append(app.cmd.extras, &c)

}

func (app App) Run(args []string, optDoRun ...bool) (tgt interface{}, tgtargs []string, appRunErr error) {
	c := &app.cmd

	doRun := true
	if len(optDoRun) > 0 && !optDoRun[0] {
		doRun = false
	}

	if len(args) == len(os.Args) && len(args) > 0 && args[0] == os.Args[0] {
		args = args[1:]
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
							fmt.Fprintf(os.Stdout, "option %s %v\n\n", string(ch), ErrNotDefined)
							app.Help(os.Stdout)
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
					fmt.Fprintf(os.Stderr, "option %s %v\n\n", name, ErrNotDefined)
					app.Help(os.Stdout)
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
					fmt.Fprintf(os.Stderr, "option %s: %v\n\n", name, err)
					app.Help(os.Stdout)
				}
				return nil, nil, err
			}

			name = ""

		} else if len(c.args) == 0 {
			if len(c.subs)+len(c.extras) > 0 {
				sub, extra := c.findSubCmd(t)
				if sub == nil {
					if !app.SuppressErrorOutput {
						fmt.Fprintf(os.Stderr, "command %s %v\n\n", t, ErrNotDefined)
						app.Help(os.Stdout)
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
				app.Help(os.Stdout)
			} else {
				c.Help(os.Stdout)
			}
		}

		return nil, nil, helpErr
	}

	// call Before->Run->After

	// Before/After
	// Before: root->sub->subsub
	// After: subsub->sub->root *deferred*

	for _, c := range cmdStack {
		callErr, beforeErr := call("Before", c.v, cmdStack, c.args)
		if callErr == nil && beforeErr != nil {
			if !app.SuppressErrorOutput {
				fmt.Fprintf(os.Stderr, "%v\n", beforeErr)
				c.Help(os.Stdout)
			}
			return nil, nil, beforeErr
		}

		if doRun {
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
				app.Help(os.Stdout)
			} else {
				c.Help(os.Stdout)
			}
			//return ErrNotDefined
			return nil, nil, nil
		}

		if runErr != nil {
			if !app.SuppressErrorOutput {
				fmt.Fprintf(os.Stderr, "%v\n", runErr)
			}
			return nil, nil, runErr
		}
	}

	return c.v.Interface(), c.args, nil
}

func gather(ttgt reflect.Type, tgt *cmd) error {
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
				if tv, ok := tag.Lookup("help"); ok && tgt.help == "" {
					tgt.help = strings.TrimSpace(tv)
				}
				// usage description
				if tv, ok := tag.Lookup("usage"); ok && tgt.usage == "" {
					tgt.usage = strings.TrimSpace(tv)
				}
			}

			continue
		}

		iscmd := false
		{ // struct is skipped if a non-Parsable
			if ft.Type.Kind() == reflect.Struct || (ft.Type.Kind() == reflect.Ptr && ft.Type.Elem().Kind() == reflect.Struct) {
				ok := ft.Type.Implements(reflect.TypeOf((*Parsable)(nil)).Elem())
				if !ok {
					ok = reflect.PtrTo(ft.Type).Implements(reflect.TypeOf((*Parsable)(nil)).Elem())
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
				fieldIdx: i,
			}
			tgt.subs = append(tgt.subs, sub)

			err := gather(ft.Type, sub)
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
		e := stack[i].v
		ei := stack[i].v.Interface()
		et := e.Type()
		if et == typ || et.Kind() == reflect.Ptr && et.Elem() == typ {
			return ei
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
}
