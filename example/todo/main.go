package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"bitbucket.org/shu_go/clise"
	"bitbucket.org/shu_go/gli"
	"bitbucket.org/shu_go/rog"
)

var (
	verbose = func(fmt string, v ...interface{}) { /* nop */ }
)

type globalCmd struct {
	List listCmd `cli:"ls,list"  help:"list todoes"  usage:"todo list [--done|--undone] [filter words...]"`
	Add  addCmd  `help:"add todoes" usage:"todo add {ITEM}...\nNote: multiple ITEMs are OK."`
	Del  delCmd  `cli:"del,delete" help:"delete todoes" usage:"todo delete [--num NUM] [filter words...]"`
	Done doneCmd `cli:"done"`

	File    string `cli:"file=FILE_PATH" default:"./todo.json" help:"file name of a storage"`
	Verbose bool   `cli:"v,verbose" help:"verbose output"`
}

type listCmd struct {
	Done   bool `cli:"done" help:"display only done items"`
	Undone bool `cli:"undone,un,u" help:"display only undone items"`
}

type addCmd struct {
	DueTo *gli.Date `cli:"due,d=DATE" help:"set due date in form of yyyy-mm-dd"`
}

type delCmd struct {
	Num  *gli.IntList `cli:"n,num=NUMBERS" help:"delete by Item Number"`
	Done bool         `cli:"done" help:"delete done items"`
}

type doneCmd struct {
	_ struct{} `help:"mark todoes as done or undone" usage:"todo done [--undone] [--num NUM] [filter words...]"`

	Num    *gli.IntList `cli:"n,num=NUMBERS" help:"delete by Item Number"`
	Undone bool         `cli:"undone,un,u" help:"mark as UNDONE (default to done)"`
}

func (g globalCmd) Before() error {
	if g.Verbose {
		verbose = func(format string, v ...interface{}) {
			fmt.Fprintf(os.Stderr, format, v...)
		}
	}
	return nil
}

func (ls listCmd) Run(global *globalCmd, args []string) error {
	if ls.Done && ls.Undone {
		//fmt.Println("--done and --undone is exclusive")
		return fmt.Errorf("--done and --undone is exclusive")
	}

	list := todoList{}
	if err := list.Load(global.File); err != nil {
		return err
	}

	if ls.Done || ls.Undone {
		done := ls.Done
		mode := "done"
		if !done {
			mode = "undone"
		}
		clise.Filter(&list, func(i int) bool {
			return list[i].Done == done
		})
		verbose("filter: %s\n\n", mode)
	}

	if len(args) != 0 {
		clise.Filter(&list, func(i int) bool {
			someok := false
			for _, a := range args {
				if strings.Contains(list[i].Content, a) {
					someok = true
				}
			}
			return someok
		})
		verbose("filter: %v\n\n", args)
	}

	if len(list) == 0 {
		fmt.Println("no matches")
	} else {
		for _, t := range list {
			fmt.Println(t)
		}
	}

	return nil
}

func (add addCmd) Run(global *globalCmd, args []string) error {
	if len(args) == 0 {
		fmt.Println("no args")
		return nil
	}

	list := todoList{}
	if err := list.Load(global.File); err != nil {
		return err
	}

	for _, c := range args {
		t := todo{
			Num:       -1,
			Content:   c,
			CreatedAt: time.Now(),
		}
		if add.DueTo != nil {
			dueto := add.DueTo.Time()
			t.DueTo = &dueto
		}
		list = append(list, t)
	}

	return list.Save(global.File)
}

func (del delCmd) Run(global *globalCmd, args []string) error {
	if del.Num == nil && len(args) == 0 && !del.Done {
		fmt.Println("no conditions")
		return nil
	}

	list := todoList{}
	if err := list.Load(global.File); err != nil {
		return err
	}

	delset := make(map[int]struct{})
	for i, t := range list {
		numsexists := false
		argsexists := false
		done := false

		if del.Num == nil {
			numsexists = true
		} else {
			if del.Num != nil && del.Num.Contains(t.Num) {
				numsexists = true
			}
		}

		if len(args) == 0 {
			argsexists = true
		} else {
			for _, a := range args {
				if strings.Contains(t.Content, a) {
					argsexists = true
				}
			}
		}

		if !del.Done {
			done = true
		} else {
			done = t.Done
		}

		if numsexists && argsexists && done {
			delset[i] = struct{}{}
		}
	}

	clise.Filter(&list, func(i int) bool {
		_, found := delset[i]
		return !found
	})

	return list.Save(global.File)
}

func (done doneCmd) Run(global *globalCmd, args []string) error {
	if done.Num == nil && len(args) == 0 {
		fmt.Println("no conditions")
		return nil
	}

	list := todoList{}
	if err := list.Load(global.File); err != nil {
		return err
	}

	mode := "done"
	if done.Undone {
		mode = "undone"
	}

	for i := 0; i < len(list); i++ {
		t := &(list)[i]
		if done.Num != nil && done.Num.Contains(t.Num) {
			verbose("%s %s\n", mode, t)
			t.Done = !done.Undone
			continue
		}
		for _, a := range args {
			if strings.Contains(t.Content, a) {
				verbose("%s %s\n", mode, t)
				t.Done = !done.Undone
				break
			}
		}
	}

	return list.Save(global.File)
}

func main() {
	rog.EnableDebug()
	//app := gli.NewWith(&globalCmd{})

	app := gli.New()
	app.OptionsGrouped = false

	app.Bind(&globalCmd{})
	app.Name = "todo"
	app.Desc = "gli example app"
	app.Version = "beta"
	app.Copyright = "(C) 2017 Shuhei Kubota"

	app.AddExtraCommand(&helloCmd{}, "hello", "say hello", gli.Usage("todo hello\nthis will greet you"))

	err := app.Run(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}
}

type helloCmd struct {
	Goodbye helloGoodbyeCmd

	Name string
}

type helloGoodbyeCmd struct {
	A, B, C bool
	ABC     bool
}

func (hello helloCmd) Run() {
	fmt.Printf("%#v\n", os.Args)
	if hello.Name != "" {
		fmt.Println("hello, " + hello.Name + "!")
	} else {
		fmt.Println("hello!")
	}
}

func (goodbye helloGoodbyeCmd) Run() {
	fmt.Println("good bye!")
	if goodbye.A {
		fmt.Println("A")
	}
	if goodbye.B {
		fmt.Println("B")
	}
	if goodbye.C {
		fmt.Println("C")
	}
	if goodbye.ABC {
		fmt.Println("ABC")
	}
}
