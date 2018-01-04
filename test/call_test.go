package test

import (
	"testing"

	"bitbucket.org/shu/gli"
	"bitbucket.org/shu/gotwant"
	"bitbucket.org/shu/rog"
)

type callGlobal struct {
	Sub    callSub
	result string
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
	rog.EnableDebug()
	defer rog.DisableDebug()

	t.Run("Sub", func(t *testing.T) {
		g := callGlobal{}
		app := gli.New(&g)
		err := app.Run([]string{"sub"})
		gotwant.Error(t, err, nil)
		gotwant.Test(t, g.result, ":global_before:sub_before:sub_run:sub_after:global_after")
	})
	t.Run("SubSub", func(t *testing.T) {
		g := callGlobal{}
		app := gli.New(&g)
		err := app.Run([]string{"sub subsub"})
		gotwant.Error(t, err, nil)
		gotwant.Test(t, g.result, ":global_before:sub_before:subsub_before:subsub_run:subsub_after:sub_after:global_after")
	})
}
