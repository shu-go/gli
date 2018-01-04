package test

import (
	"testing"
	"time"

	"bitbucket.org/shu/gli"
	"bitbucket.org/shu/gotwant"
)

func TestTypes(t *testing.T) {
	t.Run("Int", func(t *testing.T) {
		g := struct {
			Int  int
			Uint uint
		}{}
		app := gli.New(&g)
		app.Run([]string{"--int -123  --uint 321"})
		gotwant.Test(t, g.Int, -123)
		gotwant.Test(t, g.Uint, uint(321), gotwant.Format("%#v"))
	})
	t.Run("Float", func(t *testing.T) {
		g := struct {
			Float float32
		}{}
		app := gli.New(&g)
		app.Run([]string{"--float 0.25"})
		gotwant.Test(t, g.Float, float32(0.25))
	})
	t.Run("String", func(t *testing.T) {
		g := struct {
			String string
		}{}
		app := gli.New(&g)
		app.Run([]string{"--string 123"})
		gotwant.Test(t, g.String, "123")
	})
	t.Run("gli.Duration", func(t *testing.T) {
		g := struct {
			D gli.Duration
		}{}
		app := gli.New(&g)
		app.Run([]string{"-d 1m"})
		gotwant.Test(t, g.D, gli.Duration(time.Minute))
	})
	t.Run("gli.Range", func(t *testing.T) {
		g := struct {
			Range gli.Range
		}{}
		app := gli.New(&g)
		app.Run([]string{"--range 1:5"})
		gotwant.Test(t, g.Range.Min, "1")
		gotwant.Test(t, g.Range.Max, "5")
		app.Run([]string{"--range :5"})
		gotwant.Test(t, g.Range.Min, "")
		gotwant.Test(t, g.Range.Max, "5")
		app.Run([]string{"--range 1:"})
		gotwant.Test(t, g.Range.Min, "1")
		gotwant.Test(t, g.Range.Max, "")
	})
	t.Run("gli.StrList", func(t *testing.T) {
		g := struct {
			List gli.StrList
		}{}
		app := gli.New(&g)
		app.Run([]string{"--list a,b,c"})
		gotwant.Test(t, g.List, gli.StrList([]string{"a", "b", "c"}))
	})
	t.Run("multiple gli.StrList", func(t *testing.T) {
		g := struct {
			List gli.StrList
		}{}
		app := gli.New(&g)
		app.Run([]string{"--list a,b,c --list d,e,f"})
		gotwant.Test(t, g.List, gli.StrList([]string{"a", "b", "c", "d", "e", "f"}))
	})
	t.Run("ptr gli.StrList", func(t *testing.T) {
		g := struct {
			List    *gli.StrList
			NilList *gli.StrList
		}{}
		app := gli.New(&g)
		app.Run([]string{"--list a,b,c --list d,e,f"})
		gotwant.Test(t, g.List, (*gli.StrList)(&[]string{"a", "b", "c", "d", "e", "f"}))
		gotwant.Test(t, g.NilList, (*gli.StrList)(nil), gotwant.Format("%#v"))
	})
	t.Run("default ptr gli.IntList", func(t *testing.T) {
		g := struct {
			List1 *gli.IntList `default:"1,10,100"`
			List2 *gli.IntList `default:"1,10,100"`
		}{}
		app := gli.New(&g)
		app.Run([]string{"--list2 2,3,4"})
		gotwant.Test(t, g.List1, (*gli.IntList)(&[]int{1, 10, 100}))
		gotwant.Test(t, g.List2, (*gli.IntList)(&[]int{1, 10, 100, 2, 3, 4}))
	})
}
