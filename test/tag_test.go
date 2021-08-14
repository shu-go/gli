package test

import (
	"os"
	"testing"

	"github.com/shu-go/gli"
	"github.com/shu-go/gotwant"
)

func newApp(ptr interface{}) gli.App {
	app := gli.NewWith(ptr)
	app.SuppressErrorOutput = true
	app.Stdout = nil
	app.Stderr = nil
	return app
}

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

	app := newApp(&g)
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
	app = newApp(&g)
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

		app := newApp(&g)
		app.Run([]string{})
		gotwant.Test(t, g, wantg)

		g = orig
		os.Setenv("VALUE1", "zxc")
		os.Setenv("VALUE2", "-999")

		wantg = orig
		wantg.Value1 = "zxc"
		wantg.Value2 = -999
		app = newApp(&g)
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

		app := newApp(&g)
		app.Run([]string{})
		gotwant.Test(t, g, wantg)

		g = orig
		os.Setenv("VALUE1", "zxc")
		os.Setenv("VALUE2", "-999")

		wantg = orig
		wantg.Value1 = "zxc"
		wantg.Value2 = -999
		app = newApp(&g)
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

	app := newApp(&g)
	app.SuppressErrorOutput = true
	app.CliTag = "aaa"
	app.HelpTag = "bbb"
	app.UsageTag = "ccc"
	app.DefaultTag = "ddd"
	app.EnvTag = "eee"
	err := app.Bind(&g)
	gotwant.TestError(t, err, nil)

	//app.Help(os.Stdout)
	app.Run([]string{"subsub help"})

	_, _, err = app.Parse([]string{"subsub", "--v1=subsub_no_v1"})
	gotwant.TestError(t, err, nil)
	gotwant.Test(t, g.Sub.Value1, "subsub_no_v1")

	_, _, err = app.Parse([]string{"subsub"})
	gotwant.TestError(t, err, nil)
	gotwant.Test(t, g.Sub.Value1, "default")

	os.Setenv("ENV", "env")

	_, _, err = app.Parse([]string{"subsub"})
	gotwant.TestError(t, err, nil)
	gotwant.Test(t, g.Sub.Value1, "env")

	os.Setenv("ENV", "")
}

func TestRequired(t *testing.T) {
	g := struct {
		A string `cli:"a" required:"true"`
		B string `cli:"b" required:"true" default:"B"`
		C string `cli:"c"`
	}{}
	orig := g

	g = orig
	app := newApp(&g)
	err := app.Run([]string{})
	gotwant.TestError(t, err, "required")

	g = orig
	wantg := orig
	wantg.A = "ABC"
	wantg.B = "B"
	wantg.C = ""
	app = newApp(&g)
	err = app.Run([]string{"-a", "ABC"})
	gotwant.TestError(t, err, "required")

	g = orig
	wantg = orig
	wantg.A = "ABC"
	wantg.B = "B"
	wantg.C = ""
	app = newApp(&g)
	err = app.Run([]string{"-a", "ABC", "-b", "B"})
	gotwant.TestError(t, err, nil)
	gotwant.Test(t, g, wantg)
}

func TestRequiredNested(t *testing.T) {
	g := struct {
		A string `cli:"a" required:"true"`
		S struct {
			B string `cli:"b" required:"true"`
		} `cli:"s"`
	}{}
	origg := g

	app := newApp(&g)
	err := app.Run([]string{})
	gotwant.TestError(t, err, "required")

	g = origg
	wantg := origg
	wantg.A = "A"
	app = newApp(&g)
	err = app.Run([]string{"-a", "A"})
	gotwant.TestError(t, err, nil)
	gotwant.Test(t, g, wantg)

	app = newApp(&g)
	err = app.Run([]string{"-a", "A", "s"})
	gotwant.TestError(t, err, "required")

	g = origg
	wantg = origg
	wantg.A = "A"
	wantg.S.B = "B"
	app = newApp(&g)
	err = app.Run([]string{"-a", "A", "s", "-b", "B"})
	gotwant.TestError(t, err, nil)
	gotwant.Test(t, g, wantg)
}
