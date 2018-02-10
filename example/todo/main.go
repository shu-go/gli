package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"bitbucket.org/shu/clise"
	"bitbucket.org/shu/gli"
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

type addCmd struct{}

type delCmd struct {
	Num *gli.IntList `cli:"n,num=NUMBERS" help:"delete by Item Number"`
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
		list = append(list, t)
	}

	return list.Save(global.File)
}

func (del delCmd) Run(global *globalCmd, args []string) error {
	if del.Num == nil && len(args) == 0 {
		fmt.Println("no conditions")
		return nil
	}

	list := todoList{}
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
	app := gli.New(&globalCmd{})
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

type helloCmd struct{}

func (hello helloCmd) Run() {
	fmt.Println("hello!")
}
