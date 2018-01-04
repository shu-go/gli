package test

import (
	"testing"

	"bitbucket.org/shu/gli"
	"bitbucket.org/shu/gotwant"
)

func TestPtrOpt(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		g := struct {
			Int *int
			Str *string
		}{}
		app := gli.New(&g)
		app.Run([]string{""})
		gotwant.Test(t, g.Int, (*int)(nil))
		gotwant.Test(t, g.Str, (*string)(nil))
	})
	t.Run("non-nil", func(t *testing.T) {
		g := struct {
			Int *int
			Str *string
		}{}
		app := gli.New(&g)
		app.Run([]string{"--int 123 --str aaa"})
		gotwant.Test(t, *g.Int, 123)
		gotwant.Test(t, *g.Str, "aaa")
	})
}
