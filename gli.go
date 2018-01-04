package gli

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

var (
	ErrNotDefined     = fmt.Errorf("not defined")
	ErrNotRunnable    = fmt.Errorf("command not runnable")
	ErrOptCanNotBeSet = fmt.Errorf("option can not be set")
)

func setOptValue(opt reflect.Value, value string) error {
	if opt.Type().Kind() == reflect.Ptr {
		var pv reflect.Value
		if opt.IsNil() {
			pv = reflect.New(opt.Type().Elem())
		} else {
			pv = opt
		}

		err := setOptValue(pv.Elem(), value)
		if err != nil {
			return err
		}

		opt.Set(pv)
		return nil
	}
	p, ok := opt.Interface().(Parsable)
	if ok {
		return p.Parse(value)
	} else {
		p, ok := opt.Addr().Interface().(Parsable)
		if ok {
			return p.Parse(value)
		}
	}

	switch opt.Kind() {
	case reflect.String:
		opt.Set(reflect.ValueOf(value))
		return nil

	case reflect.Bool:
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		opt.Set(reflect.ValueOf(b))
		return nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		size := int(opt.Type().Size())
		i, err := strconv.ParseInt(value, 10, size*8)
		if err != nil {
			return err
		}
		opt.Set(reflect.ValueOf(i).Convert(opt.Type()))
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		size := int(opt.Type().Size())
		i, err := strconv.ParseUint(value, 10, size*8)
		if err != nil {
			return err
		}
		opt.Set(reflect.ValueOf(i).Convert(opt.Type()))
		return nil

	case reflect.Float32, reflect.Float64:
		size := int(opt.Type().Size())
		f, err := strconv.ParseFloat(value, size*8)
		if err != nil {
			return err
		}
		opt.Set(reflect.ValueOf(f).Convert(opt.Type()))
		return nil
	}

	return ErrOptCanNotBeSet
}

func setDefaultValues(c *cmd) {
	for _, o := range c.opts {
		if o.defvalue != "" {
			setOptValue(o.holder.Field(o.fieldIdx), o.defvalue)
		}
		if o.env != "" {
			envvalue := os.Getenv(o.env)
			if envvalue != "" {
				setOptValue(o.holder.Field(o.fieldIdx), envvalue)
			}
		}
	}

	for _, s := range c.extras {
		setDefaultValues(s)
	}
	for _, s := range c.subs {
		setDefaultValues(s)
	}
}
