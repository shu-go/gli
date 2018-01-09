package test

import (
	"os"
	"testing"

	"bitbucket.org/shu/gli"
	"bitbucket.org/shu/gotwant"
)

func TestDefault(t *testing.T) {
	g := struct {
		Value1 string    `default:"abc"`
		Value2 int       `default:"-123"`
		Value3 gli.Range `default:"a:z"`
		Sub    struct {
			Value4 string    `default:"def"`
			Value5 int       `default:"234"`
			Value6 gli.Range `default:":z"`
		}
	}{}
	orig := g

	wantg := orig
	wantg.Value1 = "abc"
	wantg.Value2 = -123
	wantg.Value3 = gli.Range{Min: "a", Max: "z"}

	app := gli.New(&g)
	app.Run([]string{})
	gotwant.Test(t, g, wantg)

	wantg = orig
	wantg.Value1 = "abc"
	wantg.Value2 = -123
	wantg.Value3 = gli.Range{Min: "a", Max: "z"}
	wantg.Sub.Value4 = "def"
	wantg.Sub.Value5 = 234
	wantg.Sub.Value6 = gli.Range{Min: "", Max: "z"}

	g = orig
	app = gli.New(&g)
	app.Run([]string{"sub"})
	gotwant.Test(t, g, wantg)
}

func TestEnv(t *testing.T) {
	t.Run("Env", func(t *testing.T) {
		g := struct {
			Value1 string `env:"VALUE1"`
			Value2 int    `env:"VALUE2"`
		}{}
		orig := g

		wantg := orig
		wantg.Value1 = ""
		wantg.Value2 = 0

		app := gli.New(&g)
		app.Run([]string{})
		gotwant.Test(t, g, wantg)

		g = orig
		os.Setenv("VALUE1", "zxc")
		os.Setenv("VALUE2", "-999")

		wantg = orig
		wantg.Value1 = "zxc"
		wantg.Value2 = -999
		app = gli.New(&g)
		app.Run([]string{})
		gotwant.Test(t, g, wantg)

		os.Setenv("VALUE1", "")
		os.Setenv("VALUE2", "")
	})
	t.Run("Default", func(t *testing.T) {
		g := struct {
			Value1 string `env:"VALUE1" default:"poi"`
			Value2 int    `env:"VALUE2" default:"987"`
		}{}
		orig := g

		wantg := orig
		wantg.Value1 = "poi"
		wantg.Value2 = 987

		app := gli.New(&g)
		app.Run([]string{})
		gotwant.Test(t, g, wantg)

		g = orig
		os.Setenv("VALUE1", "zxc")
		os.Setenv("VALUE2", "-999")

		wantg = orig
		wantg.Value1 = "zxc"
		wantg.Value2 = -999
		app = gli.New(&g)
		app.Run([]string{})
		gotwant.Test(t, g, wantg)

		os.Setenv("VALUE1", "")
		os.Setenv("VALUE2", "")
	})
}
