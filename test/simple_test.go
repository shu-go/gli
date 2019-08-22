package test

import (
	"testing"

	"github.com/shu-go/gotwant"
)

func TestSimple(t *testing.T) {
	t.Run("g.Name", func(t *testing.T) {
		g := struct {
			Name string `cli:"n"`
		}{}
		app := newApp(&g)
		app.Run([]string{"-n", "hoge"})
		gotwant.Test(t, g.Name, "hoge")
	})
	t.Run("g.Command.Name", func(t *testing.T) {
		g := struct {
			Command struct {
				Name string `cli:"n"`
			} `cli:"co,command"`
		}{}
		app := newApp(&g)
		app.Run([]string{"co", "-n", "hoge"})
		gotwant.Test(t, g.Command.Name, "hoge")
	})
	t.Run("ex.Name", func(t *testing.T) {
		g := struct{}{}
		ex := struct {
			Name string `cli:"n"`
		}{}
		app := newApp(&g)
		app.AddExtraCommand(&ex, "extra", "extra command")
		app.Run([]string{"extra", "-n", "hoge"})
		gotwant.Test(t, ex.Name, "hoge")
	})
}
