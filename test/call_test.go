package test

import (
	"fmt"
	"testing"

	"bitbucket.org/shu/gotwant"
)

type callGlobal struct {
	result string
	Sub    callSub
}

func (g *callGlobal) Init() {
	g.result += ":global_init"
}

func (g *callGlobal) Before() {
	g.result += ":global_before"
}

func (g *callGlobal) Run() {
	g.result += ":global_run"
}

func (g *callGlobal) After() {
	g.result += ":global_after"
}

type callSub struct {
	SubSub callSubSub
}

func (s callSub) Init(g *callGlobal) {
	g.result += ":sub_init"
}

func (s callSub) Before(g *callGlobal) {
	g.result += ":sub_before"
}

func (s callSub) Run(g *callGlobal) {
	g.result += ":sub_run"
}

func (s callSub) After(g *callGlobal) {
	g.result += ":sub_after"
}

type callSubSub struct{}

func (s callSubSub) Init(g *callGlobal) {
	g.result += ":subsub_init"
}

func (s callSubSub) Before(g *callGlobal) {
	g.result += ":subsub_before"
}

func (s callSubSub) Run(g *callGlobal) {
	g.result += ":subsub_run"
}

func (s callSubSub) After(g *callGlobal) {
	g.result += ":subsub_after"
}

func TestCall(t *testing.T) {
	t.Run("Sub", func(t *testing.T) {
		g := callGlobal{}
		app := newApp(&g)
		err := app.Run([]string{"sub"})
		gotwant.TestError(t, err, nil)
		gotwant.Test(t, g.result, ":global_init:sub_init:global_before:sub_before:sub_run:sub_after:global_after")
	})

	t.Run("SubSub", func(t *testing.T) {
		g := callGlobal{}
		app := newApp(&g)
		err := app.Run([]string{"sub", "subsub"})
		gotwant.TestError(t, err, nil)
		gotwant.Test(t, g.result, ":global_init:sub_init:subsub_init:global_before:sub_before:subsub_before:subsub_run:subsub_after:sub_after:global_after")
	})
}

type callGlobal2 struct {
	result string
	Sub1   callSub21
	Opt    string
}
type callSub21 struct {
	Sub2 callSub22
	Opt  string
}
type callSub22 struct {
	Sub3 callSub23
	Opt  string
}
type callSub23 struct {
	Opt string
}

func (s3 callSub23) Run(s1 callSub21, args []string, s2 *callSub22, g *callGlobal2) {
	g.result = fmt.Sprintf("global:%v, sub1:%v, sub2:%v, sub3:%v, args:%v", g.Opt, s1.Opt, s2.Opt, s3.Opt, args)
}

func TestRunSignature(t *testing.T) {
	g := callGlobal2{}
	app := newApp(&g)
	err := app.Run([]string{"--opt", "rei", "sub1", "--opt", "ichi", "sub2", "--opt", "ni", "sub3", "--opt", "san", "shi", "go", "roku"})
	gotwant.TestError(t, err, nil)
	gotwant.Test(t, g.result, "global:rei, sub1:ichi, sub2:ni, sub3:san, args:[shi go roku]")
}
