package gli_test

import (
	"os"
	"testing"

	"bitbucket.org/shu_go/gli"
	"bitbucket.org/shu_go/gotwant"
)

func TestParseSingle(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		global := struct{}{}
		app := gli.New(&global)
		iglobal, args, err := app.Parse([]string{})

		gotwant.TestError(t, err, nil)
		gotwant.Test(t, iglobal.(*struct{}), &global)
		gotwant.TestExpr(t, args, len(args) == 0)
	})

	t.Run("SingleNoTags", func(t *testing.T) {
		global := struct {
			Name string
			Age  int
		}{}
		app := gli.New(&global)
		_, args, err := app.Parse([]string{"--name", "hoge", "--age", "123", "a", "b"})

		gotwant.TestError(t, err, nil)
		gotwant.Test(t, global.Name, "hoge")
		gotwant.Test(t, global.Age, 123)
		gotwant.Test(t, args, []string{"a", "b"})
	})

	t.Run("SingleTags", func(t *testing.T) {
		os.Setenv("TEST_AREA", "Hashi no shita")
		global := struct {
			Name string `cli:"n"`
			Age  int    `cli:"a"`

			Country string `default:"Nihon"`
			Area1   string `env:"TEST_AREA"`
		}{}
		app := gli.New(&global)
		_, args, err := app.Parse([]string{"-n", "hoge", "-a", "123", "a", "b"})

		gotwant.TestError(t, err, nil)
		gotwant.Test(t, global.Name, "hoge")
		gotwant.Test(t, global.Age, 123)
		gotwant.Test(t, global.Country, "Nihon")
		gotwant.Test(t, global.Area1, "Hashi no shita")
		gotwant.Test(t, args, []string{"a", "b"})
	})
}

type BGlobal struct {
	Sub1 BSub1
	Sub2 BSub2 `cli:"sub2, s2"`
}

type BSub1 struct{}
type BSub2 struct {
	Int1 int
	Int2 int `cli:"int2"`
	Int3 int `cli:"i,int3"`
	Int4 int `cli:"j,int4"`

	Str1 string `default:"hoge"`
	Str2 string `env:"STR2" default:"str2"`

	Bool1 bool
}

func (sub2 BSub2) Run() {
}

func Benchmark(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g := BGlobal{}
		app := gli.New(&g)
		app.Run([]string{"sub2", "--int2", "2222", "--str2=hogehoge --bool1"})
	}
}
