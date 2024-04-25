// Package gli is a CLI parsing and mapping library.
//
//	type globalCmd struct {
//	    Name string
//	    Age  int
//	}
//	func (g *globalCmd) Run() error {
//	    // :
//	}
//	app := gli.NewWith(&globalCmd{})
//	err := app.Run(os.Args)
package gli

import (
	//"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/shu-go/cliparser"
)

var (
	// ErrNotDefined means "an option or a subcommand is not defined in the passed struct".
	ErrNotDefined = errors.New("not defined")
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

	// Copyright the app author has
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
	// DefDescTag is a tag key. default: `defdesc`
	DefDescTag string
	// EnvTag is a tag key. default: `env`
	EnvTag string
	// RequiredTag is a tag key. default: `required`
	RequiredTag string
	// DecTypeTag is a tag key that is used to lookup decoder. default: `type`
	DecTypeTag string

	// MyCommandABC => false(default): "mycommandabc" , true: "my-command-abc"
	HyphenedCommandName bool
	// MyOptionABC => false(default): "myoptionabc" , true: "my-option-abc"
	HyphenedOptionName bool
	// OptionsGrouped(default: true) allows -abc may be treated as -a -b -c.
	OptionsGrouped bool
	DoubleHyphen   bool

	// SuppressErrorOutput is an option to suppresses on cli parsing error.
	SuppressErrorOutput bool
	Stdout, Stderr      *os.File

	// true(default): bool options have --no-xxx options.
	// AutoNoBoolOptions also appends --no-xxx descriptions in help doc if .
	//
	// Options:
	//   opt1, o1  A bool option
	//   opt2, o2  A bool option (default: true)
	//     --no-opt2
	//   opt1, o1  A bool option
	AutoNoBoolOptions bool

	parser cliparser.Parser
	root   *command
}

// New makes main gli instance to parse and invoke hooks.
//
// App bridges between application logic code and CLI definition (your struct).
//
// Next, call Bind() to set the CLI up.
func New() App {
	app := App{
		parser: cliparser.New(),

		CliTag:      "cli",
		HelpTag:     "help",
		UsageTag:    "usage",
		DefaultTag:  "default",
		DefDescTag:  "defdesc",
		EnvTag:      "env",
		RequiredTag: "required",
		DecTypeTag:  "type",

		HyphenedCommandName: false,
		HyphenedOptionName:  false,
		OptionsGrouped:      true,
		AutoNoBoolOptions:   true,
		DoubleHyphen:        true,

		Stdout: os.Stdout,
		Stderr: os.Stderr,
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

	g.root = &command{
		selfV:             v,
		autoNoBoolOptions: g.AutoNoBoolOptions,
	}

	return g.scanMeta(v.Type(), g.root)
}

func (g *App) scanMeta(t reflect.Type, cmd *command) error {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	fields := fieldsOf(t)

	for i := 0; i < len(fields); i++ {
		ft := fields[i].Field

		// goto next if not public field
		if ft.Name[:1] == strings.ToLower(ft.Name[:1]) {
			if ft.Name == "help" || ft.Name == "_" {
				tag := ft.Tag

				// help description
				if tv, ok := tag.Lookup(g.HelpTag); ok && cmd.help == "" {
					cmd.help = strings.TrimSpace(tv)
				}
				// usage description
				if tv, ok := tag.Lookup(g.UsageTag); ok && cmd.usage == "" {
					cmd.usage = strings.TrimSpace(tv)
				}
			}

			continue
		}

		tag := ft.Tag

		var dectype string
		dectype = strings.TrimSpace(tag.Get(g.DecTypeTag))

		isbool := ft.Type.Kind() == reflect.Bool
		iscmd := ft.Type.Kind() == reflect.Struct && LookupTypeDecoder(ft.Type) == nil && LookupTypeDecoder(dectype) == nil ||
			ft.Type.Kind() == reflect.Ptr && ft.Type.Elem().Kind() == reflect.Struct && LookupTypeDecoder(ft.Type.Elem()) == nil && LookupTypeDecoder(dectype) == nil

		name := g.arrangeName(ft.Name, iscmd)

		names := []string{}
		var env string
		var required bool
		var defvalue string
		var defdesc string
		var help string
		var usage string
		var placeholder string

		// names
		if tv, ok := tag.Lookup(g.CliTag); ok {
			if tv == "-" || tv == "" {
				continue
			}

			clinames := strings.Split(tv, ",")
			for ni := 0; ni < len(clinames); ni++ {
				n := strings.TrimSpace(clinames[ni])
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

		/*
			if !iscmd && isbool {
				ln := names[0]
				for _, n := range names {
					if len(ln) < len(n) {
						ln = n
					}
				}
				names = append(names, "no-"+ln)
			}
		*/

		defvalue = tag.Get(g.DefaultTag)
		defdesc = strings.TrimSpace(tag.Get(g.DefDescTag))
		env = strings.TrimSpace(tag.Get(g.EnvTag))
		required, err := strconv.ParseBool(strings.TrimSpace(tag.Get(g.RequiredTag)))
		if err != nil {
			required = false
		}
		help = strings.TrimSpace(tag.Get(g.HelpTag))
		usage = strings.TrimSpace(tag.Get(g.UsageTag))

		if iscmd /* f.Kind() == reflect.Struct */ {
			sub := &command{
				names:             names,
				help:              help,
				usage:             usage,
				fieldIdx:          i,
				parent:            cmd,
				autoNoBoolOptions: g.AutoNoBoolOptions,
			}
			cmd.subs = append(cmd.subs, sub)

			//HINT
			lname := sub.longestName()
			for ni := 0; ni < len(names); ni++ {
				// a.out cmd1 cmd2
				// cmd2 [cmd1]
				g.parser.HintCommand(names[ni], cmd.longestNameStack())
				// a.out cmd1 help
				// help [cmd1]
				g.parser.HintCommand("help", cmd.longestNameStack())
				// a.out cmd1 help cmd2
				// cmd2 [cmd1 help]
				g.parser.HintCommand(names[ni], append(cmd.longestNameStack(), "help"))
				//rog.Debug("HintAlias", names[ni], lname)
				g.parser.HintAlias(names[ni], lname)
			}

			err := g.scanMeta(ft.Type, sub)
			if err != nil {
				return err
			}
		} else {
			opt := &option{
				names:              names,
				env:                env,
				defValue:           defvalue,
				defDesc:            defdesc,
				required:           required,
				dectype:            dectype,
				help:               help,
				tag:                tag,
				placeholder:        placeholder,
				fieldIdx:           fields[i].Path,
				nondefFirstParsing: true,
			}
			cmd.options = append(cmd.options, opt)

			//HINT
			lname := opt.longestName()
			for ni := 0; ni < len(names); ni++ {
				if len(names[ni]) > 1 {
					g.parser.HintLongName(names[ni], cmd.longestNameStack())
				}
				if !isbool {
					g.parser.HintWithArg(names[ni], cmd.longestNameStack())
				}
				g.parser.HintAlias(names[ni], lname)
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
	for i := 0; i < len(nameslice); i++ {
		nameslice[i] = strings.TrimSpace(nameslice[i])
		g.parser.HintCommand(nameslice[i])
		g.parser.HintCommand(nameslice[i], []string{"help"})
	}
	cmd := command{
		names:             nameslice,
		help:              help,
		selfV:             v,
		parent:            g.root,
		autoNoBoolOptions: g.AutoNoBoolOptions,
	}
	lname := cmd.longestName()
	for ni := 0; ni < len(cmd.names); ni++ {
		g.parser.HintAlias(cmd.names[ni], lname)
	}

	for _, init := range inits {
		init(&cmd)
	}

	err := g.scanMeta(v.Type(), &cmd)
	if err != nil {
		panic(err.Error())
	}

	g.root.extras = append(g.root.extras, &cmd)
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

	if !g.DoubleHyphen {
		g.parser.HintDisableDoubleHyphen()
	}

	if len(args) == len(os.Args) && len(args) > 0 && args[0] == os.Args[0] {
		args = make([]string, len(os.Args)-1)
		copy(args, os.Args[1:])
	}

	cmdStack := []*command{cmd}
	cmd.setMembersReferMe()
	cmd.setDefaultValues(g.DecTypeTag)

	_, defErr := g.call("Init", cmd.selfV, cmdStack, cmd.args)
	if defErr != nil {
		if !g.SuppressErrorOutput {
			fmt.Fprintf(g.Stderr, "%v\n", defErr)
		}
		return nil, nil, defErr
	}

	helpMode := false

	g.parser.Reset()
	g.parser.Feed(args)
	if err := g.parser.Parse(); err != nil {
		if !g.SuppressErrorOutput {
			fmt.Fprintf(g.Stderr, "%v\n", err)
		}
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
			cmd.args = append(cmd.args, c.Arg)

		case cliparser.Option:
			//rog.Debug(c.Name)
			o := cmd.findOptionExact(c.Name)

			// "--no-bool" ?
			if g.AutoNoBoolOptions && o == nil && strings.HasPrefix(c.Name, "no-") {
				o = cmd.findOptionExact(c.Name[3:])
				fv := o.ownerV.Elem().FieldByIndex(o.fieldIdx)
				if o != nil && fv.Type().Kind() == reflect.Bool {
					c.Name = c.Name[3:]
					c.Arg = "false"
				}
			}

			if o == nil {
				if !g.SuppressErrorOutput {
					fmt.Fprintf(g.Stdout, "option %q %v\n\n", c.Name, ErrNotDefined)

					var candidates []string
					for oi := 0; oi < len(cmd.options); oi++ {
						names := cmd.options[oi].names
						for ni := 0; ni < len(names); ni++ {
							if strings.HasPrefix(names[ni], c.Name) {
								candidates = append(candidates, names[ni])
								break
							} else if re, err := regexp.Compile("[" + names[ni] + "]"); err == nil {
								if len(re.ReplaceAllLiteralString(c.Name, "")) <= len(c.Name)/10 {
									candidates = append(candidates, names[ni])
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
				return nil, nil, errors.Wrap(ErrNotDefined, "option "+c.Name)
			}

			fv := o.ownerV.Elem().FieldByIndex(o.fieldIdx)
			err := setOptValue(fv, c.Arg, o.tag, g.DecTypeTag, false, &o.nondefFirstParsing)
			if err != nil {
				if !g.SuppressErrorOutput {
					fmt.Fprintf(g.Stderr, "option %q: %v\n\n", c.Name, err)
					g.Help(g.Stdout)
				}
				return nil, nil, err
			}
			o.assigned = true

		case cliparser.Command: // may be an arg
			if len(cmd.subs)+len(cmd.extras) == 0 {
				cmd.args = append(cmd.args, c.Name) // command name? -> no, it's an arg
				continue
			}

			sub, isextra := cmd.findCommandExact(c.Name)
			if sub == nil {
				if !g.SuppressErrorOutput {
					//rog.Debug("notdefined")
					fmt.Fprintf(g.Stderr, "command %q %v\n\n", c.Name, ErrNotDefined)
					g.Help(g.Stdout)
				}
				//rog.Debug("notdefined")
				return nil, nil, errors.Wrap(ErrNotDefined, "command "+c.Name)
			}

			if !isextra {
				sub.ownerV = cmd.selfV
				subt := cmd.selfV.Type().Elem().Field(sub.fieldIdx).Type
				if subt.Kind() == reflect.Ptr {
					sub.selfV = reflect.New(subt.Elem())
					cmd.selfV.Elem().Field(sub.fieldIdx).Set(sub.selfV)
				} else {
					sub.selfV = cmd.selfV.Elem().Field(sub.fieldIdx).Addr()
				}
			}
			cmd = sub
			cmdStack = append(cmdStack, cmd)
			cmd.setMembersReferMe()
			cmd.setDefaultValues(g.DecTypeTag)

			_, defErr := g.call("Init", cmd.selfV, cmdStack, cmd.args)
			if defErr != nil {
				return nil, nil, defErr
			}
		}
	}

	if helpMode {
		funcName := "Help"

		callErr, helpErr := g.call(funcName, cmd.selfV, cmdStack, cmd.args)
		if callErr == ErrNotRunnable {
			callErr, helpErr = g.call(funcName, g.root.selfV, cmdStack, g.root.args)
		}

		if callErr != nil {
			if cmd.selfV == g.root.selfV {
				g.Help(g.Stdout)
			} else {
				cmd.outputHelp(g.Stdout)
			}
		}

		return nil, nil, helpErr
	}

	err := errorIfEmptyRequired(cmdStack)
	if err != nil {
		if !g.SuppressErrorOutput {
			fmt.Fprintln(g.Stderr, err)
			g.Help(g.Stdout)
		}
		return nil, nil, err
	}

	// call Before->Run->After

	// Before/After
	// Before: root->sub->subsub
	// After: subsub->sub->root *deferred*

	if doRun {
		for ci := 0; ci < len(cmdStack); ci++ {
			callErr, beforeErr := g.call("Before", cmdStack[ci].selfV, cmdStack, cmdStack[ci].args)
			if callErr == nil && beforeErr != nil {
				if !g.SuppressErrorOutput {
					fmt.Fprintf(g.Stderr, "%v\n", beforeErr)
					cmdStack[ci].outputHelp(g.Stdout)
				}
				return nil, nil, beforeErr
			}

			defer func(cmd *command) {
				// After()
				callErr, afterErr := g.call("After", cmd.selfV, cmdStack, cmd.args)
				if callErr != nil && appRunErr == nil {
					appRunErr = afterErr
				}
			}(cmdStack[ci])
		}
	}

	if doRun {
		funcName := "Run"

		callErr, runErr := g.call(funcName, cmd.selfV, cmdStack, cmd.args)

		if callErr != nil {
			if cmd == g.root {
				g.Help(g.Stdout)
			} else {
				cmd.outputHelp(g.Stdout)
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

	return cmd.selfV.Interface(), cmd.args, nil
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

func (g *App) call(funcName string, cmd reflect.Value, cmdStack []*command, args []string) (callErr, userErr error) {
	methv := cmd.MethodByName(funcName)
	if methv == (reflect.Value{}) {
		return ErrNotRunnable, nil
	}

	var argv []reflect.Value
	for i := 0; i < methv.Type().NumIn(); i++ {
		in := methv.Type().In(i)

		if in.Kind() == reflect.Struct {
			st := findStructByType(cmdStack, in)
			if st != nil {
				argv = append(argv, reflect.ValueOf(st).Elem())
			} else if reflect.TypeOf(g) == in {
				argv = append(argv, reflect.ValueOf(*g))
			} else {
				return ErrNotRunnable, nil
			}

		} else if in.Kind() == reflect.Ptr && in.Elem().Kind() == reflect.Struct {
			st := findStructByType(cmdStack, in)
			if st != nil {
				argv = append(argv, reflect.ValueOf(st))
			} else if reflect.TypeOf(g) == in {
				argv = append(argv, reflect.ValueOf(g))
			} else {
				return ErrNotRunnable, nil
			}

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
		e := stack[i].selfV
		ei := stack[i].selfV.Interface()
		et := e.Type()
		if et == typ || et.Kind() == reflect.Ptr && et.Elem() == typ {
			return ei
		}
	}
	return nil
}

func setOptValue(opt reflect.Value, value string, tag reflect.StructTag, dectypeTag string, parsingDef bool, nondefFirstParsing *bool) error {
	if opt.Type().Kind() == reflect.Ptr {
		var pv reflect.Value
		if opt.IsNil() {
			pv = reflect.New(opt.Type().Elem())
		} else {
			pv = opt
		}

		err := setOptValue(pv.Elem(), value, tag, dectypeTag, parsingDef, nondefFirstParsing)
		if err != nil {
			return err
		}

		opt.Set(pv)
		return nil
	}

	ndfp := *nondefFirstParsing
	if !parsingDef && *nondefFirstParsing {
		*nondefFirstParsing = false
	}

	dec := LookupTypeDecoder(strings.TrimSpace(tag.Get(dectypeTag)))
	if dec != nil {
		return dec(value, opt, tag, ndfp)
	} else {
		dec = LookupTypeDecoder(opt.Type())
		if dec != nil {
			err := dec(value, opt, tag, ndfp)
			if err == nil {
				return nil
			}
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

	default:
		//nop
	}

	return ErrOptCanNotBeSet
}

func errorIfEmptyRequired(cmdStack []*command) error {
	for i := len(cmdStack) - 1; i >= 0; i-- {
		c := cmdStack[i]
		for _, o := range c.options {
			if !o.required {
				continue
			}

			if !o.assigned {
				return errors.New("option " + o.longestName() + " is required")
			}
		}
	}

	return nil
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

	g.root.usage = g.Usage

	g.root.outputHelp(w)

	fmt.Fprintln(w, `
Help sub commands:
  help     `+g.Name+` help subcommnad subsubcommand
  version  show version`)

	if g.Copyright != "" {
		fmt.Fprintf(w, "\n%s\n", g.Copyright)
	}
}

type fieldAndPath struct {
	Field reflect.StructField
	Path  []int
}

func fieldsOf(t reflect.Type) []fieldAndPath {
	result := make([]fieldAndPath, 0, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		if ft.Type.Kind() == reflect.Struct && ft.Anonymous {
			for j := 0; j < ft.Type.NumField(); j++ {
				jft := ft.Type.Field(j)
				result = append(result, fieldAndPath{Field: jft, Path: []int{i, j}})
			}
		} else {
			result = append(result, fieldAndPath{Field: ft, Path: []int{i}})
		}
	}

	return result
}
