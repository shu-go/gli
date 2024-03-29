package test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/shu-go/gli/v2"
	"github.com/shu-go/gotwant"
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
	t.Run("*Int", func(t *testing.T) {
		g := struct {
			Int  *int
			Uint *uint
		}{}
		app := newApp(&g)
		err := app.Run([]string{"--int", "-123", "--uint", "321"})
		gotwant.TestError(t, err, nil)
		gotwant.Test(t, *g.Int, -123)
		gotwant.Test(t, *g.Uint, uint(321), gotwant.Format("%#v"))
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
	t.Run("*String", func(t *testing.T) {
		g := struct {
			String *string
		}{}
		app := newApp(&g)
		app.Run([]string{"--string", "123"})
		gotwant.Test(t, *g.String, "123")
	})
	t.Run("time.Duration", func(t *testing.T) {
		g := struct {
			D time.Duration
		}{}
		app := newApp(&g)
		app.Run([]string{"-d", "1m"})
		gotwant.Test(t, g.D, time.Minute)
	})
	t.Run("*time.Duration", func(t *testing.T) {
		g := struct {
			D *time.Duration
		}{}
		app := newApp(&g)
		err := app.Run([]string{"-d", "1m"})
		gotwant.TestError(t, err, nil)
		gotwant.Test(t, *g.D, time.Minute)
	})
	t.Run("time.Time", func(t *testing.T) {
		g := struct {
			T time.Time
		}{}
		app := newApp(&g)
		err := app.Run([]string{"-t", "2019/01/31"})
		fmt.Fprintf(os.Stderr, "err: %+v\n", err)
		gotwant.TestError(t, err, nil)
		gotwant.Test(t, g.T, time.Date(2019, 1, 31, 0, 0, 0, 0, time.Local))
	})
	t.Run("*time.Time", func(t *testing.T) {
		g := struct {
			T *time.Time
		}{}
		app := newApp(&g)
		err := app.Run([]string{"-t", "2019/01/31"})
		fmt.Fprintf(os.Stderr, "err: %+v\n", err)
		gotwant.TestError(t, err, nil)
		gotwant.Test(t, *g.T, time.Date(2019, 1, 31, 0, 0, 0, 0, time.Local))
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
	t.Run("StrSlice", func(t *testing.T) {
		g := struct {
			List []string `default:"d,e,f"`
		}{}
		app := newApp(&g)
		app.Run([]string{"--list", "a,b,c"})
		gotwant.Test(t, g.List, []string{"a", "b", "c"})
	})
	t.Run("NG StrSlice", func(t *testing.T) {
		g := struct {
			List []string
		}{}
		app := newApp(&g)
		app.Run([]string{"--list", "a,b,", "c"})
		gotwant.Test(t, g.List, []string{"a", "b"})

		g = struct {
			List []string
		}{}
		app = newApp(&g)
		app.Run([]string{"--list", "a,b", ",c"})
		gotwant.Test(t, g.List, []string{"a", "b"})
	})
	t.Run("multiple StrSlice", func(t *testing.T) {
		g := struct {
			List []string `default:"d,e,f"`
		}{}
		app := newApp(&g)
		app.Run([]string{"--list", "a,b,c", "--list", "d,e,f"})
		gotwant.Test(t, g.List, []string{"a", "b", "c", "d", "e", "f"})
	})
	t.Run("ptr StrSlice", func(t *testing.T) {
		g := struct {
			List    *[]string
			NilList *[]string
		}{}
		app := newApp(&g)
		app.Run([]string{"--list", "a,b,c", "--list", "d,e,f"})
		gotwant.Test(t, g.List, &[]string{"a", "b", "c", "d", "e", "f"})
		gotwant.Test(t, g.NilList, (*[]string)(nil), gotwant.Format("%#v"))
	})
	t.Run("default ptr []int", func(t *testing.T) {
		g := struct {
			List1 *[]int `default:"1,10,100"`
			List2 *[]int `default:"1,10,100"`
		}{}
		app := newApp(&g)
		app.Run([]string{"--list2", "2,3,4"})
		gotwant.Test(t, g.List1, (*[]int)(&[]int{1, 10, 100}))
		gotwant.Test(t, g.List2, (*[]int)(&[]int{2, 3, 4}))
	})
	t.Run("map[string]string", func(t *testing.T) {
		g := struct {
			Map1 map[string]string `cli:"D"`
			Map2 map[string]string `cli:"E" default:"a:b, c:d"`
		}{}
		app := newApp(&g)
		app.Run([]string{})
		gotwant.Test(t, g.Map1, (map[string]string)(nil))
		gotwant.Test(t, g.Map2, map[string]string{"a": "b", "c": "d"})

		app = newApp(&g)
		app.Run([]string{"-D", `hoge:hogehoge`, "-D", "moge:mogemoge"})
		gotwant.Test(t, g.Map1, map[string]string{"hoge": "hogehoge", "moge": "mogemoge"})

		app = newApp(&g)
		app.Run([]string{"-D", `hoge:ho\, geh:oge`})
		gotwant.Test(t, g.Map1, map[string]string{"hoge": "ho, geh:oge"})

		app = newApp(&g)
		app.Run([]string{"-E", `a:`, "-E", "moge:mogemoge"})
		gotwant.Test(t, g.Map2, map[string]string{"moge": "mogemoge"})
	})
}
