package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"bitbucket.org/shu/clise"
	"bitbucket.org/shu/gli"
	"bitbucket.org/shu/rog"
)

var (
	verbose = func(fmt string, v ...interface{}) { /* nop */ }
)

type TodoGlobal struct {
	List TodoListCmd `cli:"ls,list"  help:"list todoes"  usage:"todo list [--done|--undone] [filter words...]"`
	Add  TodoAddCmd  `help:"add todoes" usage:"todo add {ITEM}...\nNote: multiple ITEMs are OK."`
	Del  TodoDelCmd  `cli:"del,delete" help:"delete todoes" usage:"todo delete [--num NUM] [filter words...]"`
	Done TodoDoneCmd `cli:"done"`

	File    string `cli:"file=FILE_PATH" default:"./todo.json" help:"file name of a storage"`
	Verbose bool   `cli:"v,verbose" help:"verbose output"`
}

type TodoListCmd struct {
	Done   bool `cli:"done" help:"display only done items"`
	Undone bool `cli:"undone,un,u" help:"display only undone items"`
}

type TodoAddCmd struct{}

type TodoDelCmd struct {
	Num *gli.IntList `cli:"n,num=NUMBERS" help:"delete by Item Number"`
}

type TodoDoneCmd struct {
	help struct{} `help:"mark todoes as done or undone" usage:"todo done [--undone] [--num NUM] [filter words...]"`

	Num    *gli.IntList `cli:"n,num=NUMBERS" help:"delete by Item Number"`
	Undone bool         `cli:"undone,un,u" help:"mark as UNDONE (default to done)"`
}

func (g TodoGlobal) Before() error {
	if g.Verbose {
		verbose = func(format string, v ...interface{}) {
			fmt.Fprintf(os.Stderr, format, v...)
		}
		rog.EnableDebug()
	}
	return nil
}

func (ls TodoListCmd) Run(global *TodoGlobal, args []string) error {
	if ls.Done && ls.Undone {
		//fmt.Println("--done and --undone is exclusive")
		return fmt.Errorf("--done and --undone is exclusive")
	}

	list := TodoList{}
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

func (add TodoAddCmd) Run(global *TodoGlobal, args []string) error {
	if len(args) == 0 {
		fmt.Println("no args")
		return nil
	}

	list := TodoList{}
	if err := list.Load(global.File); err != nil {
		return err
	}

	for _, c := range args {
		t := Todo{
			Num:       -1,
			Content:   c,
			CreatedAt: time.Now(),
		}
		list = append(list, t)
	}

	if err := list.Save(global.File); err != nil {
		return err
	}

	return nil
}

func (del TodoDelCmd) Run(global *TodoGlobal, args []string) error {
	if del.Num == nil && len(args) == 0 {
		fmt.Println("no conditions")
		return nil
	}

	list := TodoList{}
	if err := list.Load(global.File); err != nil {
		return err
	}

	clise.Filter(&list, func(i int) bool {
		t := (list)[i]
		if del.Num != nil && del.Num.Contains(t.Num) {
			verbose("delete %s\n", t)
			return false
		}
		for _, a := range args {
			if strings.Contains(t.Content, a) {
				verbose("delete %s\n", t)
				return false
			}
		}
		return true
	})

	if err := list.Save(global.File); err != nil {
		return err
	}

	return nil
}

func (done TodoDoneCmd) Run(global *TodoGlobal, args []string) error {
	if done.Num == nil && len(args) == 0 {
		fmt.Println("no conditions")
		return nil
	}

	list := TodoList{}
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

	if err := list.Save(global.File); err != nil {
		return err
	}

	return nil
}

func main() {

	app := gli.New(&TodoGlobal{})
	app.Name = "todo"
	app.Desc = "gli example app"
	app.Version = "beta"
	app.Copyright = "(C) 2017 Shuhei Kubota"

	app.AddExtraCommand(&Hello{}, "hello", "say hello", gli.Usage("todo hello\nthis will greet you"))

	app.Run(os.Args)
}

type Hello struct{}

func (hello Hello) Run() {
	fmt.Println("hello!")
}
