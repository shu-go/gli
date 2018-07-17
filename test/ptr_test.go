package test

import (
	"testing"

	"bitbucket.org/shu_go/gotwant"
)

func TestPtrOpt(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		g := struct {
			Int *int
			Str *string
		}{}
		app := newApp(&g)
		app.Run([]string{""})
		gotwant.Test(t, g.Int, (*int)(nil))
		gotwant.Test(t, g.Str, (*string)(nil))
	})
	t.Run("non-nil", func(t *testing.T) {
		g := struct {
			Int *int
			Str *string
		}{}
		app := newApp(&g)
		app.Run([]string{"--int", "123", "--str", "aaa"})
		gotwant.Test(t, *g.Int, 123)
		gotwant.Test(t, *g.Str, "aaa")
	})
}

type PtrGlobal struct {
	Sub1 *PtrSub1
	Sub2 *PtrSub2
}

type PtrSub1 struct {
	Opt1 string
}

type PtrSub2 struct {
	Opt1 string
}

func (c *PtrSub1) Run() {
	subresult = c.Opt1
}

func (c *PtrSub2) Run() {
	subresult = c.Opt1
}

func TestPtrSubcmd(t *testing.T) {
	g := PtrGlobal{}
	app := newApp(&g)
	app.Run([]string{"sub1", "--opt1", "abc"})
	gotwant.Test(t, subresult, "abc")
	gotwant.Test(t, g.Sub1.Opt1, "abc")
	gotwant.Test(t, g.Sub2, (*PtrSub2)(nil))
}
