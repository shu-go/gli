package test

import (
	"testing"

	"bitbucket.org/shu/gli"
	"bitbucket.org/shu/gotwant"
	"bitbucket.org/shu/rog"
)

type quotedGlobal struct {
	Quoted quoted
}
type quoted struct {
	Opt    string
	result []string
}

func (q *quoted) Run(args []string) {
	if q.Opt != "" {
		q.result = append(q.result, "Opt:"+q.Opt)
	}

	for _, v := range args {
		q.result = append(q.result, v)
	}
}

func TestQuoted(t *testing.T) {
	t.Run("ParseArg", func(t *testing.T) {
		rog.EnableDebug()

		c := quotedGlobal{}
		app := gli.New(&c)
		_, args, err := app.Parse([]string{"quoted a b c d"})
		gotwant.TestError(t, err, nil)
		gotwant.Test(t, args, []string{"a", "b", "c", "d"})

		c = quotedGlobal{}
		app = gli.New(&c)
		_, args, _ = app.Parse([]string{`quoted a b "c d"`})
		gotwant.Test(t, args, []string{"a", "b", "c d"})

		c = quotedGlobal{}
		app = gli.New(&c)
		_, args, _ = app.Parse([]string{"quoted abc", `"def ghi"`, "jkl"})
		gotwant.Test(t, args, []string{"abc", `def ghi`, "jkl"})

		rog.DisableDebug()
	})

	t.Run("RunArg", func(t *testing.T) {
		c := quotedGlobal{}
		app := gli.New(&c)
		app.Run([]string{"quoted a b c d"})
		gotwant.Test(t, c.Quoted.result, []string{"a", "b", "c", "d"})

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Run([]string{`quoted a b "c d"`})
		gotwant.Test(t, c.Quoted.result, []string{"a", "b", "c d"})

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Run([]string{"quoted abc", `"def ghi"`, "jkl"})
		gotwant.Test(t, c.Quoted.result, []string{"abc", `def ghi`, "jkl"})
	})

	t.Run("ParseOpt", func(t *testing.T) {
		c := quotedGlobal{}
		app := gli.New(&c)
		app.Parse([]string{"quoted --opt", "abc"})
		gotwant.Test(t, c.Quoted.Opt, "abc")

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Parse([]string{"quoted --opt", `def ghi`})
		gotwant.Test(t, c.Quoted.Opt, "def")

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Parse([]string{"quoted --opt", `"def ghi"`})
		gotwant.Test(t, c.Quoted.Opt, "def ghi")

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Parse([]string{`quoted --opt="def ghi"`})
		gotwant.Test(t, c.Quoted.Opt, "def ghi")
	})

	t.Run("RunOpt", func(t *testing.T) {
		c := quotedGlobal{}
		app := gli.New(&c)
		app.Run([]string{"quoted --opt", "abc"})
		gotwant.Test(t, c.Quoted.result, []string{"Opt:abc"})

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Run([]string{"quoted --opt", `def ghi`})
		gotwant.Test(t, c.Quoted.result, []string{"Opt:def", "ghi"})

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Run([]string{"quoted --opt", `"def ghi"`})
		gotwant.Test(t, c.Quoted.result, []string{"Opt:def ghi"})

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Run([]string{`quoted --opt="def ghi"`})
		gotwant.Test(t, c.Quoted.result, []string{"Opt:def ghi"})
	})

	t.Run("RunBoth", func(t *testing.T) {
		c := quotedGlobal{}
		app := gli.New(&c)
		app.Run([]string{"quoted --opt", `"def ghi" "j k  l"`})
		gotwant.Test(t, c.Quoted.result, []string{"Opt:def ghi", "j k  l"})
	})
}
