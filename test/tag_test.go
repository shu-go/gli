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
	app.SuppressErrorOutput = true
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

func TestTagName(t *testing.T) {
	g := struct {
		Sub *struct {
			_      struct{} `bbb:"help of subsub" ccc:"usage of subsub"`
			Value1 string   `aaa:"v1" bbb:"help" ccc:"usage" ddd:"default" eee:"ENV"`
		} `aaa:"subsub"`
	}{}

	app := gli.New(&g)
	app.SuppressErrorOutput = true
	app.CliTag = "aaa"
	app.HelpTag = "bbb"
	app.UsageTag = "ccc"
	app.DefaultTag = "ddd"
	app.EnvTag = "eee"
	err := app.Rescan(&g)
	gotwant.Error(t, err, nil)

	app.Help(os.Stdout)
	app.Run([]string{"subsub help"})

	_, _, err = app.Run([]string{"subsub --v1=subsub_no_v1"}, false)
	gotwant.Error(t, err, nil)
	gotwant.Test(t, g.Sub.Value1, "subsub_no_v1")

	_, _, err = app.Run([]string{"subsub"}, false)
	gotwant.Error(t, err, nil)
	gotwant.Test(t, g.Sub.Value1, "default")

	os.Setenv("ENV", "env")
	_, _, err = app.Run([]string{"subsub"}, false)
	gotwant.Error(t, err, nil)
	gotwant.Test(t, g.Sub.Value1, "env")
}
