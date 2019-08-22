# features

* struct base
* tag (`cli:"names, n" help:"help message" default:"parsable literal"`) 
  * for sub commands (cli, help, usage)
  * for options (cli, help, default, env)
* sub command as a member struct in a parent struct
  * sub sub ... command
* extra sub command
* user defined option types (example: gli.Range, gli.IntList, ...)
* pointer type options
* hook functions Init/Before/Run/After/Help as methods of commands

# go get

> go get github.com/shu-go/gli

example app:

> go get github.com/shu-go/gli/example/todo

This introduces an executable binary `todo`.

# Examples

## Example1: Simple

```go
type Global struct {
    Opt1 string
    Opt2 int
}

func main() {
    app := gli.New(&Global{})
    _, _, err := app.Run(os.Args)
    // :
}

func (g *Global) Run(args []string) {
}

// app --opt1 abc --opt2 123
// app --opt1=abc --opt2=123
```

## Example2: Renaming

```go
type Global struct {
    Opt1 string `cli:"s, str"`
    Opt2 int    `cli:"i, int, opt2"`
}

// :

// app --opt1 abc --opt2 123 <-- NG: opt1 is not defined (while opt2 is defined)
// app -s abc -i 123
// app --str abc --int 123
```

## Example3: Sub command

```go
type Global struct {
    Opt1 string `cli:"s, str"`
    Opt2 int    `cli:"i, int, opt2"`

    Sub1 mySub
}

type mySub struct {
    Opt3 string
    
    Sub2 mySubSub `cli:s, sub2`
}

func (sub *mySub) Run(g *Global, args []string) {
}

func (subsub *mySubSub) Run(g *Global, args []string, sub *mySub) {
}

// app --str abc --int 123 sub1 --opt3 def
```

## Example4: Hook functions

Commnds (root and sub commands) may have some hook functions.

Define receivers for the target commands.

```go
func (subsub *mySubSub) Run(g *Global, args []string, sub *mySub) {
}
```

* Run
  * is called for the target command
* Before
  * are called for root -> sub -> subsub(target, in this case) 
  * With any error, Run is not called.
* After
  * are called for subsub(target, in this case) -> sub -> root
* Init
  * are called for root -> sub -> subsub(target, in this case) 
  * These functions are for initialization of command struct.
* Help
  * prints help message.
  * subsub(target, in this case) -> root

### Run

1. Init for all commands
2. Before for all commands
   * and defer calling After
3. Run

### Help

1. Init for all commands
2. first defined Help, subsub -> root

### Signature

Parameters are in arbitrary order, omittable.

* `[]string`
* `struct{...}` or `*struct{...}` of command

```go
// OK
func (subsub *mySubSub) Run(args []string, g *Global, sub *mySub) error {
}
// OK
func (subsub *mySubSub) Run(g *Global, args []string, sub *mySub) error {
}
// OK
func (subsub *mySubSub) Run(args []string, sub *mySub) error {
}
// OK
func (subsub *mySubSub) Run(sub *mySub) error {
}
```

Return value is nothing or an error.

```go
func (subsub *mySubSub) Run() {
}
func (subsub *mySubSub) Run() error {
}
```

## Example5: No Hook

Using gli to get values. No Run() implemented.

```go
type Global struct {
    Opt1 string
    Opt2 int
    Sub1 *Sub1Cmd
}

type Sub1Cmd struct {
    Opt3 string
}

func main() {
    g := Global{}
    app := gli.New(&g)
    tgt, tgtargs, err := app.Run(os.Args, false) // no hook

    // traverse g
    println(g.Opt1) // abc
    println(g.Opt2) // 123
    if g.Sub1 != nil {
        println(g.Sub1.Opt3) // def
    }

    if sub1, ok := tgt.(*Sub1Cmd); ok {
        println(sub1.Opt3) // def
    }
    println(tgtargs) // g h i
}

func (g *Global) Run(args []string) {
    // not called
}

// app --opt1 abc --opt2 123 sub1 --opt3 def  g h i
```



## Example6: Extra Command

```go
type Global struct {}

func main() {
    ex := extra{
        Name string `cli:"n"`
    }{}

    app := gli.New(&Global{})
    app.AddExtraCommand(&ex, "extra", "help message")
}

// app extra -n abc
```

## Example7: User defined option types

```go
type MyOption struct {
    Data int
}

func (o *MyOption) Parse(s string) error {
    o.Data = len(s)
}

//

type Global struct {
    My MyOption
}
```

## Example8: more tags

```go
type Global struct {
    Opt1 string `cli:"opt1=PLACE_HOLDER" default:"default value" env:"ENV_OPT1" help:"help message"`
    Sub MySub   `cli:"sub" help:"help message" usage:"multi line usages\nseparated by \\n"`
}
```

Options:
* cli
  * renaming
  * `cli:"name1, name2, ..., nameZ=PLACE_HOLDER"`
* default
  * in string literal
  * bool, float, int and uint are converted by strconv.ParseXXX
  * other types are required implement func Parse (see Example7)
  * use Init hook function for dynamic default values
* env
  * environment variable name
* help

Sub commands:
* cli
* help
* usage
  * multi line usage description separated by \n.

Option value overwriting:
1. default tag
2. env tag
3. Init hook function

## Example9: alternative help and usage of commands

```go
type Global struct {
    Sub1 SubCommand1 `cli:"s1"  help:"a command"  usage:"s1 [anything]"`
    Sub2 SubCommand2 `cli:"s2"` // no help and usage
}

type SubCommand2 struct {
    help struct{} `help:"another command" usage:"s2 [something]"`

    // Underscore is also OK.
    //_ struct{} `help:"another command" usage:"s2 [something]"` 
}
```

Both Sub1 and Sub2 are handled as have same tags.

----

Copyright 2018 Shuhei Kubota

<!--  vim: set et ft=markdown sts=4 sw=4 ts=4 tw=0 : -->
