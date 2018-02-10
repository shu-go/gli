package test

import (
	"testing"

	"bitbucket.org/shu/gli"
	"bitbucket.org/shu/gotwant"
)

type nohookGlobal struct {
	Sub1 *nohookSub1
	Sub2 *nohookSub2
	Opt0 string
}

type nohookSub1 struct {
	Sub12 *nohookSub12
	Opt1  string
}

type nohookSub2 struct {
	Opt2 string
}

type nohookSub12 struct {
	Opt12 string
}

type nohookExtra0 struct {
	ExtOpt0 string
	ExtSub1 *nohookExtra1
	ExtSub2 *nohookExtra2
}

type nohookExtra1 struct {
	ExtOpt1 string
}

type nohookExtra2 struct {
	ExtOpt2 string
}

func TestNoHook(t *testing.T) {
	orig := nohookGlobal{}

	g := orig
	app := gli.New(&g)
	tgt, tgtargs, err := app.Parse([]string{"--opt0 abc", "sub1 --opt1 def", "sub12 --opt12 ghi j k l"})

	gotwant.Test(t, err, nil)
	if sub12, ok := tgt.(*nohookSub12); ok {
		gotwant.Test(t, *sub12, nohookSub12{Opt12: "ghi"})
	} else {
		t.Errorf("tgt is not *nohookSub12 but %T", tgt)
	}
	gotwant.Test(t, tgtargs, []string{"j", "k", "l"})

	gotwant.Test(t, g.Opt0, "abc")
	gotwant.Test(t, g.Sub1.Opt1, "def")
	gotwant.Test(t, g.Sub1.Sub12.Opt12, "ghi")
	gotwant.Test(t, g.Sub2, (*nohookSub2)(nil))

	g = orig
	e := nohookExtra0{}
	app.AddExtraCommand(&e, "extra", "extra usage")
	tgt, _, _ = app.Parse([]string{"--opt0 abc", "extra --extopt0 def", "extsub1 --extopt1 ghi j k l"})
	if extsub1, ok := tgt.(*nohookExtra1); ok {
		gotwant.Test(t, *extsub1, nohookExtra1{ExtOpt1: "ghi"})
	} else {
		t.Fatalf("tgt is not *nohookExtra1 but %T", tgt)
	}
	gotwant.Test(t, e.ExtOpt0, "def")
	gotwant.Test(t, e.ExtSub1.ExtOpt1, "ghi")
	gotwant.Test(t, e.ExtSub2, (*nohookExtra2)(nil))
}
