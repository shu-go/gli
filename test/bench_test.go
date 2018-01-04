package test

import (
	"testing"

	"bitbucket.org/shu/gli"
)

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
	Str2 string `env:"STR2" defualt:"str2"`

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
