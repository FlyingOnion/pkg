package sqlwrapper

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"time"
)

type Strategy uint8

const (
	DoNothing Strategy = iota
	SetZero
	Continue
)

type ValueConverter interface {
	TimeFormat() string
	ConvertString(dptr interface{}, src string) error
	ConvertInt64(dptr interface{}, src int64) error
	ConvertTime(dptr interface{}, src time.Time, format string) error
	ConvertBool(dptr interface{}, src bool) error
	ConvertBytes(dptr interface{}, src []byte) error
	ConvertFloat64(dptr interface{}, src float64) error
}

const (
	DefaultTimeFormat = "2006-01-02 15:04:05"
)

type vcie struct{}

var Vcie = vcie{}

func (vcie) TimeFormat() string { return DefaultTimeFormat }

func (c vcie) ConvertString(dptr interface{}, src string) error {
	switch d := dptr.(type) {
	case nil:
		return ErrNilPointer
	case *string:
		*d = src
	case *[]byte:
		*d = []byte(src)
	case *sql.RawBytes:
		*d = append((*d)[:0], src...)
	case *sql.NullString:
		d.Valid = true
		d.String = src
	// may have error
	case *sql.NullInt64:
		err := c.ConvertString(&d.Int64, src)
		d.Valid = err == nil
		return err
	case *sql.NullInt32:
		err := c.ConvertString(&d.Int32, src)
		d.Valid = err == nil
		return err
	case *sql.NullInt16:
		err := c.ConvertString(&d.Int16, src)
		d.Valid = err == nil
		return err
	case *sql.NullFloat64:
		f64, err := strconv.ParseFloat(src, 64)
		if err != nil {
			return fmt.Errorf(f2, src, src, reflect.TypeOf(dptr).Elem().String(), err)
		}
		d.Valid = true
		d.Float64 = f64
	case *sql.NullTime:
		t, err := time.ParseInLocation(c.TimeFormat(), src, time.Local)
		if err != nil {
			return fmt.Errorf(f2, src, src, reflect.TypeOf(dptr).Elem().String(), err)
		}
		d.Valid = true
		d.Time = t
	case *sql.NullBool:
		b, err := strconv.ParseBool(src)
		if err != nil {
			return fmt.Errorf(f2, src, src, reflect.TypeOf(dptr).Elem().String(), err)
		}
		d.Valid = true
		d.Bool = b
	default:
		goto reflectConversion
	}
	return nil
reflectConversion:
	dtype := reflect.TypeOf(dptr).Elem()
	dv := reflect.ValueOf(dptr).Elem()

	switch dtype.Kind() {
	case reflect.Ptr:
		d := dv.Interface()
		if d != nil {
			return c.ConvertString(d, src)
		}
		ev := reflect.New(dtype.Elem())
		err := c.ConvertString(ev.Interface(), src)
		if err == nil {
			dv.Set(ev)
		}
		return err
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i64, err := strconv.ParseInt(src, 10, dtype.Bits())
		if err != nil {
			return fmt.Errorf(f2, src, src, dtype.String(), err)
		}
		dv.SetInt(i64)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u64, err := strconv.ParseUint(src, 10, dtype.Bits())
		if err != nil {
			return fmt.Errorf(f2, src, src, dtype.String(), err)
		}
		dv.SetUint(u64)
	case reflect.Float32, reflect.Float64:
		f64, err := strconv.ParseFloat(src, dtype.Bits())
		if err != nil {
			return fmt.Errorf(f2, src, src, dtype.String(), err)
		}
		dv.SetFloat(f64)
	case reflect.Bool:
		b, err := strconv.ParseBool(src)
		if err != nil {
			return fmt.Errorf(f2, src, src, dtype.String(), err)
		}
		dv.SetBool(b)
	case reflect.String:
		dv.SetString(src)
	default:
		goto unsupported
	}
	return nil
unsupported:
	return fmt.Errorf(f1, src, dtype.String())
}

func (c vcie) ConvertInt64(dptr interface{}, src int64) error {
	switch d := dptr.(type) {
	case nil:
		return ErrNilPointer
	case *int64:
		*d = src
	case *int:
		*d = int(src)
	case *string:
		*d = strconv.FormatInt(src, 10)
	case *[]byte:
		*d = strconv.AppendInt((*d)[:0], src, 10)
	case *sql.RawBytes:
		*d = strconv.AppendInt((*d)[:0], src, 10)
	case *sql.NullInt64:
		d.Valid = true
		d.Int64 = src
	case *sql.NullInt32:
		err := c.ConvertInt64(&d.Int32, src)
		d.Valid = err == nil
		return err
	case *sql.NullInt16:
		err := c.ConvertInt64(&d.Int16, src)
		d.Valid = err == nil
		return err
	case *sql.NullFloat64:
		d.Valid = true
		d.Float64 = float64(src)
	case *sql.NullString:
		d.Valid = true
		d.String = strconv.FormatInt(src, 10)
	default:
		goto reflectConversion
	}
	return nil

reflectConversion:
	dtype := reflect.TypeOf(dptr).Elem()
	dv := reflect.ValueOf(dptr).Elem()

	switch dtype.Kind() {
	case reflect.Ptr:
		d := dv.Interface()
		if d != nil {
			return c.ConvertInt64(d, src)
		}
		ev := reflect.New(dtype.Elem())
		err := c.ConvertInt64(ev.Interface(), src)
		if err == nil {
			dv.Set(ev)
		}
		return err
	case reflect.Int64, reflect.Int:
		dv.SetInt(src)
	case reflect.Uint64:
		dv.SetUint(uint64(src))
	case reflect.Int8, reflect.Int16, reflect.Int32:
		bound := int64(1<<dtype.Bits() - 1)
		if src|bound != bound {
			return fmt.Errorf(f2, src, src, dtype.String(), "value out of range")
		}
		dv.SetInt(src)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		bound := int64(1<<dtype.Bits() - 1)
		if src|bound != bound {
			return fmt.Errorf(f2, src, src, dtype.String(), "value out of range")
		}
		dv.SetUint(uint64(src))
	case reflect.Float32, reflect.Float64:
		dv.SetFloat(float64(src))
	case reflect.String:
		dv.SetString(strconv.FormatInt(src, 10))
	case reflect.Bool:
		if src == 0 || src == 1 {
			dv.SetBool(src == 1)
			return nil
		}
		return fmt.Errorf(f2, src, src, dtype.String(), "only 0 and 1 could convert to bool")
	default:
		goto unsupported
	}
	return nil
unsupported:
	return fmt.Errorf(f1, src, dtype.String())
}

func (c vcie) ConvertTime(dptr interface{}, src time.Time, format string) error {
	switch d := dptr.(type) {
	case nil:
		return ErrNilPointer
	case *time.Time:
		*d = src
	case *string:
		*d = src.Format(format)
	case *[]byte:
		*d = src.AppendFormat((*d)[:0], format)
	case *sql.RawBytes:
		*d = src.AppendFormat((*d)[:0], format)
	case *sql.NullTime:
		d.Valid = true
		d.Time = src
	case *sql.NullString:
		d.Valid = true
		d.String = src.Format(format)
	default:
		goto reflectConversion
	}
	return nil
reflectConversion:
	dtype := reflect.TypeOf(dptr).Elem()
	dv := reflect.ValueOf(dptr).Elem()

	switch dtype.Kind() {
	case reflect.Ptr:
		d := dv.Interface()
		if d != nil {
			return c.ConvertTime(d, src, format)
		}
		ev := reflect.New(dtype.Elem())
		err := c.ConvertTime(ev.Interface(), src, format)
		if err == nil {
			dv.Set(ev)
		}
		return err
	case reflect.String:
		dv.SetString(src.Format(format))
	default:
		goto unsupported
	}
	return nil
unsupported:
	return fmt.Errorf(f1, src, dtype.String())
}

func (c vcie) ConvertBool(dptr interface{}, src bool) error {
	switch d := dptr.(type) {
	case nil:
		return ErrNilPointer
	case *bool:
		*d = src
	case *string:
		*d = strconv.FormatBool(src)
	case *[]byte:
		*d = strconv.AppendBool((*d)[:0], src)
	case *sql.RawBytes:
		*d = strconv.AppendBool((*d)[:0], src)
	case *sql.NullBool:
		d.Valid = true
		d.Bool = src
	case *sql.NullString:
		d.Valid = true
		d.String = strconv.FormatBool(src)
	default:
		goto reflectConversion
	}
	return nil
reflectConversion:
	dtype := reflect.TypeOf(dptr).Elem()
	dv := reflect.ValueOf(dptr).Elem()

	switch dtype.Kind() {
	case reflect.Ptr:
		d := dv.Interface()
		if d != nil {
			return c.ConvertBool(d, src)
		}
		ev := reflect.New(dtype.Elem())
		err := c.ConvertBool(ev.Interface(), src)
		if err == nil {
			dv.Set(ev)
		}
		return err
	case reflect.Bool:
		dv.SetBool(src)
	case reflect.String:
		dv.SetString(strconv.FormatBool(src))
	default:
		goto unsupported
	}
	return nil
unsupported:
	return fmt.Errorf(f1, src, dtype.String())
}

func (c vcie) ConvertBytes(dptr interface{}, src []byte) error {
	switch d := dptr.(type) {
	case nil:
		return ErrNilPointer
	case *[]byte:
		*d = cloneBytes(src)
	case *sql.RawBytes:
		*d = cloneBytes(src)
	case *string:
		*d = string(src)
	case *sql.NullString:
		d.Valid = true
		d.String = string(src)
	default:
		goto reflectConversion
	}
	return nil
reflectConversion:
	dtype := reflect.TypeOf(dptr).Elem()
	dv := reflect.ValueOf(dptr).Elem()

	switch dtype.Kind() {
	case reflect.Ptr:
		d := dv.Interface()
		if d != nil {
			return c.ConvertBytes(d, src)
		}
		ev := reflect.New(dtype.Elem())
		err := c.ConvertBytes(ev.Interface(), src)
		if err == nil {
			dv.Set(ev)
		}
		return err
	case reflect.String:
		dv.SetString(string(src))
	default:
		goto stringConversion
	}
	return nil
stringConversion:
	if err := c.ConvertString(dptr, string(src)); err == nil {
		return nil
	}
	// unsupported
	return fmt.Errorf(f1, src, dtype.String())
}

func (c vcie) ConvertFloat64(dptr interface{}, src float64) error {
	switch d := dptr.(type) {
	case nil:
		return ErrNilPointer
	case *float64:
		*d = src
	case *float32:
		*d = float32(src)
	case *string:
		*d = strconv.FormatFloat(src, 'f', -1, 64)
	case *[]byte:
		*d = strconv.AppendFloat((*d)[:0], src, 'f', -1, 64)
	case *sql.RawBytes:
		*d = strconv.AppendFloat((*d)[:0], src, 'f', -1, 64)
	case *sql.NullFloat64:
		d.Valid = true
		d.Float64 = src
	case *sql.NullString:
		d.Valid = true
		d.String = strconv.FormatFloat(src, 'f', -1, 64)
	case *sql.NullInt64:
		err := c.ConvertFloat64(&d.Int64, src)
		d.Valid = err == nil
		return err
	case *sql.NullInt32:
		err := c.ConvertFloat64(&d.Int32, src)
		d.Valid = err == nil
		return err
	case *sql.NullInt16:
		err := c.ConvertFloat64(&d.Int16, src)
		d.Valid = err == nil
		return err
	default:
		goto reflectConversion
	}
	return nil
reflectConversion:
	dtype := reflect.TypeOf(dptr).Elem()
	dv := reflect.ValueOf(dptr).Elem()

	switch dtype.Kind() {
	case reflect.Ptr:
		d := dv.Interface()
		if d != nil {
			return c.ConvertFloat64(d, src)
		}
		ev := reflect.New(dtype.Elem())
		err := c.ConvertFloat64(ev.Interface(), src)
		if err == nil {
			dv.Set(ev)
		}
		return err
	case reflect.Float32, reflect.Float64:
		dv.SetFloat(src)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		str := strconv.FormatFloat(src, 'f', -1, 64)
		i64, err := strconv.ParseInt(str, 10, dtype.Bits())
		if err != nil {
			return fmt.Errorf(f2, src, str, dtype.String(), err)
		}
		dv.SetInt(i64)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		str := strconv.FormatFloat(src, 'f', -1, 64)
		u64, err := strconv.ParseUint(str, 10, dtype.Bits())
		if err != nil {
			return fmt.Errorf(f2, src, str, dtype.String(), err)
		}
		dv.SetUint(u64)
	case reflect.String:
		dv.SetString(strconv.FormatFloat(src, 'f', -1, 64))
	default:
		goto unsupported
	}
	return nil
unsupported:
	return fmt.Errorf(f1, src, dtype.String())
}

func convertValue(
	dptr, src interface{},
	converter ValueConverter,
	onNull Strategy,
) error {
	if dptr == nil {
		return ErrNilPointer
	}

	switch s := src.(type) {
	case nil:
		switch onNull {
		case DoNothing:
			return nil
		case SetZero:
			dv := reflect.ValueOf(dptr).Elem()
			dv.Set(reflect.Zero(dv.Type()))
			return nil
		default:
		}
	case int64:
		return converter.ConvertInt64(dptr, s)
	case string:
		return converter.ConvertString(dptr, s)
	case []byte:
		return converter.ConvertBytes(dptr, s)
	case time.Time:
		return converter.ConvertTime(dptr, s, converter.TimeFormat())
	case float64:
		return converter.ConvertFloat64(dptr, s)
	case bool:
		return converter.ConvertBool(dptr, s)
	}
	return fmt.Errorf(f3, src)
}

func cloneBytes(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
