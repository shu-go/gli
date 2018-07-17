package test

import (
	"testing"

	"bitbucket.org/shu_go/gli"
	"bitbucket.org/shu_go/gotwant"
)

type quotedGlobal struct {
	Quoted quoted
}

type quoted struct {
	Opt    string
	result []string
}

type singles struct {
	A, B, C bool
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

		c := quotedGlobal{}
		app := gli.New(&c)
		_, args, err := app.Parse([]string{"quoted", "a", "b", "c", "d"})
		gotwant.TestError(t, err, nil)
		gotwant.Test(t, args, []string{"a", "b", "c", "d"})

		c = quotedGlobal{}
		app = gli.New(&c)
		_, args, _ = app.Parse([]string{"quoted", "a", "b", "c d"})
		gotwant.Test(t, args, []string{"a", "b", "c d"})

		c = quotedGlobal{}
		app = gli.New(&c)
		_, args, _ = app.Parse([]string{"quoted", "abc", "def ghi", "jkl"})
		gotwant.Test(t, args, []string{"abc", `def ghi`, "jkl"})
	})

	t.Run("RunArg", func(t *testing.T) {
		c := quotedGlobal{}
		app := gli.New(&c)
		app.Run([]string{"quoted", "a", "b", "c", "d"})
		gotwant.Test(t, c.Quoted.result, []string{"a", "b", "c", "d"})

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Run([]string{"quoted", "a", "b", "c d"})
		gotwant.Test(t, c.Quoted.result, []string{"a", "b", "c d"})

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Run([]string{"quoted", "abc", "def ghi", "jkl"})
		gotwant.Test(t, c.Quoted.result, []string{"abc", `def ghi`, "jkl"})
	})

	t.Run("ParseOpt", func(t *testing.T) {
		c := quotedGlobal{}
		app := gli.New(&c)
		app.Parse([]string{"quoted", "--opt", "abc"})
		gotwant.Test(t, c.Quoted.Opt, "abc")

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Parse([]string{"quoted", "--opt", "def", "ghi"})
		gotwant.Test(t, c.Quoted.Opt, "def")

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Parse([]string{"quoted", "--opt", "def ghi"})
		gotwant.Test(t, c.Quoted.Opt, "def ghi")

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Parse([]string{"quoted", "--opt=def ghi"})
		gotwant.Test(t, c.Quoted.Opt, "def ghi")
	})

	t.Run("RunOpt", func(t *testing.T) {
		c := quotedGlobal{}
		app := gli.New(&c)
		app.Run([]string{"quoted", "--opt", "abc"})
		gotwant.Test(t, c.Quoted.result, []string{"Opt:abc"})

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Run([]string{"quoted", "--opt", "def", "ghi"})
		gotwant.Test(t, c.Quoted.result, []string{"Opt:def", "ghi"})

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Run([]string{"quoted", "--opt", "def ghi"})
		gotwant.Test(t, c.Quoted.result, []string{"Opt:def ghi"})

		c = quotedGlobal{}
		app = gli.New(&c)
		app.Run([]string{"quoted", "--opt=def ghi"})
		gotwant.Test(t, c.Quoted.result, []string{"Opt:def ghi"})
	})

	t.Run("RunBoth", func(t *testing.T) {
		c := quotedGlobal{}
		app := gli.New(&c)
		app.Run([]string{"quoted", "--opt", "def ghi", "j k  l"})
		gotwant.Test(t, c.Quoted.result, []string{"Opt:def ghi", "j k  l"})
	})

	t.Run("Singles", func(t *testing.T) {
		c := singles{}
		app := gli.New(&c)
		app.Run([]string{"-a", "-b", "-c"})
		gotwant.Test(t, c.A, true)
		gotwant.Test(t, c.B, true)
		gotwant.Test(t, c.C, true)
	})

	t.Run("SinglesConcat", func(t *testing.T) {
		c := singles{}
		app := gli.New(&c)
		app.Run([]string{"-ac"})
		gotwant.Test(t, c.A, true)
		gotwant.Test(t, c.B, false)
		gotwant.Test(t, c.C, true)
	})

	t.Run("SinglesConcat+", func(t *testing.T) {
		c := singles{}
		app := gli.New(&c)
		app.Run([]string{"-ac", "-b"})
		gotwant.Test(t, c.A, true)
		gotwant.Test(t, c.B, true)
		gotwant.Test(t, c.C, true)
	})
}

func TestHyphen(t *testing.T) {
	t.Run("NotAShortOpt", func(t *testing.T) {
		c := struct {
			A int
			B int
		}{}
		app := gli.New(&c)
		app.Run([]string{"-a=-1", "-b", "-2"})
		gotwant.Test(t, c.A, -1)
		gotwant.Test(t, c.B, -2)
	})

	t.Run("NotALongOpt", func(t *testing.T) {
		c := struct {
			A string
			B string
			C int
		}{}
		app := gli.New(&c)
		app.Run([]string{"-a=--1", "-b", "--2", "-c", "-3"})
		gotwant.Test(t, c.A, "--1")
		gotwant.Test(t, c.B, "--2")
		gotwant.Test(t, c.C, -3)
	})
}
