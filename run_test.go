package gli_test

import (
	"strings"
	"testing"

	"bitbucket.org/shu/gli"
	"bitbucket.org/shu/gotwant"
)

var subresult string

type Sub1 struct{}

func (s Sub1) Run() {
	subresult = "sub1"
}

type Sub2 struct{ Name string }

func (s Sub2) Run() {
	subresult = "sub2" + s.Name
}

type Sub3 struct{ Name string }

func (s Sub3) Run(g *struct {
	Sub3 Sub3
	Name string
}) {
	subresult = g.Name + "sub3" + s.Name
}

type Sub4 struct{ Name string }

func (s Sub4) Run(g *struct {
	Sub4Holder struct {
		Sub4 Sub4
		Name string
	}
	Name string
}) {
	subresult = g.Name + "sub4" + s.Name
}

func TestCommandRun(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		g := struct {
			Sub1 Sub1
		}{}
		app := gli.New(&g)
		app.Run([]string{"sub1"})
		gotwant.Test(t, subresult, "sub1")
	})
	t.Run("Opt", func(t *testing.T) {
		g := struct {
			Sub2 Sub2
		}{}
		app := gli.New(&g)
		app.Run([]string{"sub2 --name namae"})
		gotwant.Test(t, subresult, "sub2namae")
	})
	t.Run("Global", func(t *testing.T) {
		g := struct {
			Sub3 Sub3
			Name string
		}{}
		app := gli.New(&g)
		app.Run([]string{"--name guro-baru sub3 --name namae"})
		gotwant.Test(t, subresult, "guro-barusub3namae")
	})
	t.Run("Nest", func(t *testing.T) {
		g := struct {
			Sub4Holder struct {
				Sub4 Sub4
				Name string
			}
			Name string
		}{}
		app := gli.New(&g)
		app.Run([]string{"--name guro-baru sub4holder --name holder sub4 --name namae4"})
		gotwant.Test(t, subresult, "guro-barusub4namae4")
	})
}

type Sub51 struct{ Name string }

func (s Sub51) Run() { subresult = "sub51" }

type Sub52 struct{ Name string }

func (s Sub52) Run(args []string) { subresult = "sub52" + strings.Join(args, " ") }

type Sub53 struct{ Name string }

func (s Sub53) Run(g *struct {
	Sub  Sub53
	Name string
}, args []string) {
	subresult = g.Name + "sub53" + strings.Join(args, " ")
}

type Sub54 struct{ Name string }

func (s Sub54) Run(args []string, g *struct {
	Sub  Sub54
	Name string
}) {
	subresult = g.Name + "sub54" + strings.Join(args, " ")
}

type Sub55 struct{ Name string }

func (s Sub55) Run(g *struct {
	Sub  Sub55
	Name string
}) {
	subresult = g.Name + "sub55"
}

func TestCommandRunArgs(t *testing.T) {
	t.Run("None", func(t *testing.T) {
		g := struct {
			Sub Sub51
		}{}
		app := gli.New(&g)
		app.Run([]string{"sub a b c"})
		gotwant.Test(t, subresult, "sub51")
	})
	t.Run("Args", func(t *testing.T) {
		g := struct {
			Sub Sub52
		}{}
		app := gli.New(&g)
		app.Run([]string{"sub a b c"})
		gotwant.Test(t, subresult, "sub52a b c")
	})
	t.Run("GlobalArgs", func(t *testing.T) {
		g := struct {
			Sub  Sub53
			Name string
		}{}
		app := gli.New(&g)
		app.Run([]string{"--name global sub a b c"})
		gotwant.Test(t, subresult, "globalsub53a b c")
	})
	t.Run("ArgsGlobal", func(t *testing.T) {
		g := struct {
			Sub  Sub54
			Name string
		}{}
		app := gli.New(&g)
		app.Run([]string{"--name global sub a b c"})
		gotwant.Test(t, subresult, "globalsub54a b c")
	})
	t.Run("Global", func(t *testing.T) {
		g := struct {
			Sub  Sub55
			Name string
		}{}
		app := gli.New(&g)
		app.Run([]string{"--name global sub a b c"})
		gotwant.Test(t, subresult, "globalsub55")
	})
}
