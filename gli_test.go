package gli

import (
	"testing"

	"bitbucket.org/shu/gotwant"
)

func init() {
	//rog.EnableDebug()
}

func TestStructure1(t *testing.T) {
	o := struct {
		Option1 string
	}{}
	app := New(&o)
	app.Stdout = nil
	app.Stderr = nil
	gotwant.Test(t, app.cmd.opts[0].names, []string{"option1"}, gotwant.Format("%#v"))

	oo := struct {
		Option1 string `cli:"o1,o2"`
	}{}
	app = New(&oo)
	app.Stdout = nil
	app.Stderr = nil
	gotwant.Test(t, app.cmd.opts[0].names, []string{"o1", "o2"}, gotwant.Format("%#v"))

	ooo := struct {
		Option1 string `env:"ENVIRON1" default:"hogehoge"`
	}{}
	app = New(&ooo)
	app.Stdout = nil
	app.Stderr = nil
	gotwant.Test(t, app.cmd.opts[0].names, []string{"option1"}, gotwant.Format("%#v"))
	gotwant.Test(t, app.cmd.opts[0].env, "ENVIRON1", gotwant.Format("%#v"))
	gotwant.Test(t, app.cmd.opts[0].defvalue, "hogehoge", gotwant.Format("%#v"))
}

func TestStructure2(t *testing.T) {
	o := struct {
		Sub1 struct {
			Option1 bool
		}
	}{}
	app := New(&o)
	app.Stdout = nil
	app.Stderr = nil
	gotwant.Test(t, len(app.cmd.opts), 0, gotwant.Format("%#v"))
	gotwant.Test(t, app.cmd.subs[0].names, []string{"sub1"}, gotwant.Format("%#v"))
	gotwant.Test(t, len(app.cmd.subs[0].opts), 1, gotwant.Format("%#v"))
	gotwant.Test(t, app.cmd.subs[0].opts[0].names, []string{"option1"}, gotwant.Format("%#v"))
}

func TestStructure3(t *testing.T) {
	o := struct {
		Sub1 *struct {
			Option1 bool
		}
	}{}
	app := New(&o)
	app.Stdout = nil
	app.Stderr = nil
	gotwant.Test(t, len(app.cmd.opts), 0, gotwant.Format("%#v"))
	gotwant.Test(t, app.cmd.subs[0].names, []string{"sub1"}, gotwant.Format("%#v"))
	gotwant.Test(t, len(app.cmd.subs[0].opts), 1, gotwant.Format("%#v"))
	gotwant.Test(t, app.cmd.subs[0].opts[0].names, []string{"option1"}, gotwant.Format("%#v"))
}

func TestRun1(t *testing.T) {
	o := struct {
		Name    string
		Verbose bool
	}{}
	app := New(&o)
	app.Stdout = nil
	app.Stderr = nil
	app.Run([]string{"--name=ichi"})
	gotwant.Test(t, o.Name, "ichi")

	app.Run([]string{"--name", "ni"})
	gotwant.Test(t, o.Name, "ni")

	app.Run([]string{"--name=", "san"})
	gotwant.Test(t, o.Name, "san")

	app.Run([]string{"--name", "shi", "--verbose"})
	gotwant.Test(t, o.Name, "shi")
	gotwant.Test(t, o.Verbose, true)
}

func TestRun2(t *testing.T) {
	o := struct {
		Name    string `cli:"n,name"`
		Verbose bool   `cli:"v,verbose"`
	}{}
	inito := o

	app := New(&o)
	app.Stdout = nil
	app.Stderr = nil
	app.Run([]string{"--name=ichi"})
	gotwant.Test(t, o.Name, "ichi")

	o = inito
	app.Run([]string{"--name", "ni", "--verbose"})
	gotwant.Test(t, o.Name, "ni")
	gotwant.Test(t, o.Verbose, true)

	o = inito
	app.Run([]string{"-vn", "san"})
	gotwant.Test(t, o.Name, "san")
	gotwant.Test(t, o.Verbose, true)

	o = inito
	app.Run([]string{"-n", "shi"})
	gotwant.Test(t, o.Name, "shi")
	gotwant.Test(t, o.Verbose, false)

	o = inito
	err := app.Run([]string{"-verbose"})
	gotwant.Test(t, o.Verbose, true)
	gotwant.TestError(t, err, nil)

	o = inito
	err = app.Run([]string{"-varbose"})
	gotwant.TestError(t, err, ErrNotDefined)
}
