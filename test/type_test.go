package test

import (
	"testing"
	"time"

	"bitbucket.org/shu_go/gli"
	"bitbucket.org/shu_go/gotwant"
)

func TestTypes(t *testing.T) {
	t.Run("Int", func(t *testing.T) {
		g := struct {
			Int  int
			Uint uint
		}{}
		app := newApp(&g)
		err := app.Run([]string{"--int", "-123", "--uint", "321"})
		gotwant.TestError(t, err, nil)
		gotwant.Test(t, g.Int, -123)
		gotwant.Test(t, g.Uint, uint(321), gotwant.Format("%#v"))
	})
	t.Run("Float", func(t *testing.T) {
		g := struct {
			Float float32
		}{}
		app := newApp(&g)
		app.Run([]string{"--float", "0.25"})
		gotwant.Test(t, g.Float, float32(0.25))
	})
	t.Run("String", func(t *testing.T) {
		g := struct {
			String string
		}{}
		app := newApp(&g)
		app.Run([]string{"--string", "123"})
		gotwant.Test(t, g.String, "123")
	})
	t.Run("gli.Duration", func(t *testing.T) {
		g := struct {
			D gli.Duration
		}{}
		app := newApp(&g)
		app.Run([]string{"-d", "1m"})
		gotwant.Test(t, g.D, gli.Duration(time.Minute))
	})
	t.Run("gli.Range", func(t *testing.T) {
		g := struct {
			Range gli.Range
		}{}
		app := newApp(&g)
		app.Run([]string{"--range", "1:5"})
		gotwant.Test(t, g.Range.Min, "1")
		gotwant.Test(t, g.Range.Max, "5")
		app.Run([]string{"--range", ":5"})
		gotwant.Test(t, g.Range.Min, "")
		gotwant.Test(t, g.Range.Max, "5")
		app.Run([]string{"--range", "1:"})
		gotwant.Test(t, g.Range.Min, "1")
		gotwant.Test(t, g.Range.Max, "")
	})
	t.Run("gli.StrList", func(t *testing.T) {
		g := struct {
			List gli.StrList
		}{}
		app := newApp(&g)
		app.Run([]string{"--list", "a,b,c"})
		gotwant.Test(t, g.List, gli.StrList([]string{"a", "b", "c"}))
	})
	t.Run("NG gli.StrList", func(t *testing.T) {
		g := struct {
			List gli.StrList
		}{}
		app := newApp(&g)
		app.Run([]string{"--list", "a,b,", "c"})
		gotwant.Test(t, g.List, gli.StrList([]string{"a", "b", ""}))

		g = struct {
			List gli.StrList
		}{}
		app = newApp(&g)
		app.Run([]string{"--list", "a,b", ",c"})
		gotwant.Test(t, g.List, gli.StrList([]string{"a", "b"}))
	})
	t.Run("multiple gli.StrList", func(t *testing.T) {
		g := struct {
			List gli.StrList
		}{}
		app := newApp(&g)
		app.Run([]string{"--list", "a,b,c", "--list", "d,e,f"})
		gotwant.Test(t, g.List, gli.StrList([]string{"d", "e", "f"}))
	})
	t.Run("ptr gli.StrList", func(t *testing.T) {
		g := struct {
			List    *gli.StrList
			NilList *gli.StrList
		}{}
		app := newApp(&g)
		app.Run([]string{"--list", "a,b,c", "--list", "d,e,f"})
		gotwant.Test(t, g.List, (*gli.StrList)(&[]string{"d", "e", "f"}))
		gotwant.Test(t, g.NilList, (*gli.StrList)(nil), gotwant.Format("%#v"))
	})
	t.Run("default ptr gli.IntList", func(t *testing.T) {
		g := struct {
			List1 *gli.IntList `default:"1,10,100"`
			List2 *gli.IntList `default:"1,10,100"`
		}{}
		app := newApp(&g)
		app.Run([]string{"--list2", "2,3,4"})
		gotwant.Test(t, g.List1, (*gli.IntList)(&[]int{1, 10, 100}))
		gotwant.Test(t, g.List2, (*gli.IntList)(&[]int{2, 3, 4}))
	})
	t.Run("gli.Map", func(t *testing.T) {
		g := struct {
			Map1 gli.Map `cli:"D"`
			Map2 gli.Map `cli:"E" default:"a:b, c:d"`
		}{}
		app := newApp(&g)
		app.Run([]string{})
		gotwant.Test(t, g.Map1, (gli.Map)(nil))
		gotwant.Test(t, g.Map2, (gli.Map)(map[string]string{"a": "b", "c": "d"}))

		app = newApp(&g)
		app.Run([]string{"-D", `"hoge:hogehoge"`, "-D", "moge:mogemoge"})
		gotwant.Test(t, g.Map1, (gli.Map)(map[string]string{"hoge": "hogehoge", "moge": "mogemoge"}))

		app = newApp(&g)
		app.Run([]string{"-E", `"a:"`, "-E", "moge:mogemoge"})
		gotwant.Test(t, g.Map2, (gli.Map)(map[string]string{"c": "d", "moge": "mogemoge"}))
	})
}
