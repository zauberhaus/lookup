// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package lookup

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	utils "github.com/zauberhaus/reflect_utils"
	"github.com/zauberhaus/slice_utils"
)

var (
	array = regexp.MustCompile(`(.*)\[(.*)\]`)
)

type mode int

const (
	get mode = iota
	set
	exists
	create
)

func Exists(obj any, path string) (bool, error) {
	parts := split(path)

	v := reflect.ValueOf(obj)

	if !utils.IsStruct(v) {
		return false, fmt.Errorf("exists supports only structs")
	}

	result, err := process(v, exists, nil, nil, parts...)
	if err != nil {
		return false, err
	}

	return !utils.IsNil(result), nil
}

func Get(obj any, path string) (any, error) {
	parts := split(path)

	v := reflect.ValueOf(obj)

	if !utils.IsStruct(v) {
		return nil, fmt.Errorf("get supports only structs")
	}

	return process(v, get, nil, nil, parts...)
}

func Create(obj any, path string) (any, error) {
	parts := split(path)

	v := reflect.ValueOf(obj)

	if !utils.IsStruct(v) {
		return nil, fmt.Errorf("create supports only structs")
	}

	if !utils.IsPointer(v) {
		return nil, fmt.Errorf("create supports only struct pointers")
	}

	return process(v, create, nil, nil, parts...)
}

// Set sets the field at path to value. A string is parsed into the type of
// the field; hooks parse it into types Parse doesn't know, such as an
// interface type.
func Set(obj any, path string, value any, hooks ...ParserHook) (any, error) {

	parts := split(path)

	v := reflect.ValueOf(obj)

	if !utils.IsStruct(v) {
		return nil, fmt.Errorf("set supports only structs")
	}

	if !utils.IsPointer(v) {
		return nil, fmt.Errorf("set supports only struct pointers")
	}

	return process(v, set, value, hooks, parts...)
}

func process(v reflect.Value, mode mode, value any, hooks []ParserHook, path ...string) (any, error) {
	if len(path) == 0 {
		return nil, nil
	}

	last := len(path) == 1

	for v.Kind() == reflect.Pointer {
		v = v.Elem()
	}

	if !utils.IsStruct(v) {
		return nil, fmt.Errorf("field isn't a struct")
	}

	fn := strings.ToLower(path[0])
	fn = strings.Trim(fn, " \t\n\r")

	key := any(nil)

	matches := array.FindStringSubmatch(fn)
	if len(matches) == 3 {
		fn = matches[1]
		key = matches[2]

	}

	var f reflect.Value
	t := v.Type()

	found := false
	var fi int
	for i := 0; i < v.NumField(); i++ {
		value := v.Field(i)
		field := t.Field(i)
		if strings.ToLower(field.Name) == fn {
			if !field.IsExported() {
				return nil, fmt.Errorf("field %v is not exported", fn)
			}

			f = value
			fi = i
			found = true
			break
		}
	}

	if !found {
		return nil, &NotFoundError{fn}
	}

	t = f.Type()

	var val any

	var err error

	field := v.Field(fi)

	if mode == set && last && key == nil {
		tp := v.Type().Field(fi)
		f := reflect.ValueOf(value)

		if field.CanSet() {
			if (f == reflect.Value{}) {
				t := field.Type()
				f = reflect.Zero(t)
				field.Set(f)
				val = f.Interface()
			} else {
				if field.Type().Kind() != reflect.Interface {
					if field.Type().Kind() == reflect.Pointer {
						if f.Kind() != reflect.Pointer {
							value = utils.CopyToHeap(value)
							f = reflect.ValueOf(value)
						}
					} else {
						if f.Kind() == reflect.Pointer {
							if f.IsNil() {
								return nil, fmt.Errorf("can't set a nil %v to a %v field", f.Type(), field.Type())
							}

							value = utils.FromPointer(value)
							f = reflect.ValueOf(value)
						}
					}
				}

				if field.Type().Kind() == reflect.Interface {
					if f.Type().Implements(field.Type()) {
						field.Set(f)
						val = value
					} else if txt, ok := hookInput(f, field.Type(), hooks); ok {
						val, err = parseInterface(txt, field.Type(), hooks)
						if err != nil {
							return nil, err
						}

						field.Set(reflect.ValueOf(val))
					} else {
						return nil, fmt.Errorf("%v doesn't implement %v", f.Type(), field.Type())
					}
				} else if f.Kind() == reflect.String {
					val, err = Parse(f.String(), field.Type(), hooks...)
					if err != nil {
						return nil, err
					}

					field.Set(reflect.ValueOf(val))
				} else if f.Kind() == reflect.Pointer && f.Elem().Kind() == reflect.String {
					val, err = Parse(f.Elem().String(), field.Type(), hooks...)
					if err != nil {
						return nil, err
					}

					field.Set(reflect.ValueOf(val))
				} else {
					if canConvert(f, field.Type()) {
						f = f.Convert(field.Type())
						value = f.Interface()
					} else if field.Type() != f.Type() || (field.Type().Kind() == reflect.Interface && !f.Type().Implements(field.Type())) {
						return nil, fmt.Errorf("invalid data type %v for %v field", f.Type(), field.Type())
					}

					field.Set(f)
					val = value

				}
			}
		} else {
			return nil, fmt.Errorf("field isn't addressable: %v", tp.Name)
		}
	} else if utils.IsNil(f) && (!last || mode == create) {
		if mode == exists {
			return nil, nil
		}

		val, err = utils.NewWithDefaultsOf(t)
		if err != nil {
			return nil, err
		}

		f = reflect.ValueOf(val)

		// get continues on the new value without adding it to the object
		if mode != get {
			field := v.Field(fi)

			if field.CanSet() {
				field.Set(f)
			} else {
				return nil, fmt.Errorf("field isn't addressable: %v", field)
			}
		}
	} else {
		val = f.Interface()
	}

	index := -1
	if key != nil && (f.Kind() == reflect.Slice || f.Kind() == reflect.Array) {
		if len(key.(string)) == 0 {
			index = f.Len()
			key = nil
		} else {
			if idx, err := strconv.Atoi(key.(string)); err == nil {
				if idx < 0 {
					return nil, fmt.Errorf("invalid index %v", idx)
				}

				index = idx
				key = nil
			}
		}
	}

	if index >= 0 {
		if f.Kind() == reflect.Array {
			if f.Len() <= index && (mode == get || mode == exists) {
				if mode == exists {
					return nil, nil
				}

				val, f, err = newElem(f.Type().Elem())
				if err != nil {
					return nil, err
				}
			} else if f.Len() > index {
				i := f.Index(index)
				if mode == set && last {
					val, _, err = setValue(i, i.Type(), value, hooks, func(field, f reflect.Value) {
						field.Set(f)
					})
					if err != nil {
						return nil, err
					}
				} else {
					if i.CanInterface() {
						val = i.Interface()
					} else {
						return nil, fmt.Errorf("field isn't accessible: %v", i)
					}
				}
			} else {
				return nil, errors.New("array isn't expandable")
			}
		} else if f.Kind() == reflect.Slice && f.Len() <= index && (mode == get || mode == exists) {
			if mode == exists {
				return nil, nil
			}

			val, f, err = newElem(t.Elem())
			if err != nil {
				return nil, err
			}
		} else if f.Kind() == reflect.Slice {
			l := f.Len()
			e := t.Elem()

			for i := 0; i <= index; i++ {
				if i >= l {
					_, elem, err := newElem(e)
					if err != nil {
						return nil, err
					}

					f = reflect.Append(f, elem)
				}

				if i == index {
					i := f.Index(index)

					if mode == set && last {
						val, _, err = setValue(i, i.Type(), value, hooks, func(field, f reflect.Value) {
							field.Set(f)
						})

						if err != nil {
							return nil, err
						}
					} else {
						if i.CanInterface() {
							val = i.Interface()
						}
					}
				}
			}

			field := v.Field(fi)
			if field.CanSet() {
				field.Set(f)
			} else {
				return nil, fmt.Errorf("field isn't addressable: %v", field)
			}

			f = f.Index(index)
			if f.CanInterface() {
				val = f.Interface()
			}

		}
	} else if key != nil {
		if f.Kind() != reflect.Map {
			return nil, ErrNotMap
		}

		// check if map is nil
		if utils.IsNil(f) {
			if mode == exists {
				return nil, nil
			}

			val, err = utils.NewWithDefaultsOf(t)
			if err != nil {
				return nil, err
			}

			val = utils.CopyToHeap(val)
			f = reflect.ValueOf(val).Elem()

			// get looks up the key in the new map without adding it to the object
			if mode != get {
				field := v.Field(fi)

				if field.CanSet() {
					field.Set(f)
				} else {
					return nil, fmt.Errorf("field isn't addressable: %v", field)
				}
			}
		}

		if txt, ok := key.(string); ok {
			key = strings.Trim(txt, "\"\\`'")
		}

		// check if key must be parsed
		e := f.Type().Key()
		k := reflect.ValueOf(key)

		if e.Kind() != k.Kind() {
			if e.Kind() != reflect.String && k.Kind() == reflect.String {
				val, err := Parse(k.String(), e, hooks...)
				if err != nil {
					return nil, err
				}

				key = val
				k = reflect.ValueOf(key)
			}
		}

		i := f.MapIndex(k)

		if mode == set && last {
			if utils.IsNil(value) {
				f.SetMapIndex(k, reflect.Value{})
				val = nil
				f = reflect.Value{}
			} else {
				e := f.Type().Elem()

				val, f, err = setValue(f, e, value, hooks, func(field, value reflect.Value) {
					field.SetMapIndex(k, value)
				})

				if err != nil {
					return nil, err
				}
			}
		} else {
			if i.IsValid() {
				val = i.Interface()
				f = i
			} else {
				if mode == exists {
					return nil, nil
				}

				m := f

				val, f, err = newElem(m.Type().Elem())
				if err != nil {
					return nil, err
				}

				if mode != get {
					m.SetMapIndex(k, f)
				}
			}
		}
	}

	if last {
		return val, nil
	}

	rest := path[1:]
	return process(f, mode, value, hooks, rest...)
}

func setValue(field reflect.Value, e reflect.Type, value any, hooks []ParserHook, set func(field reflect.Value, value reflect.Value)) (result any, v reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			}
		}
	}()

	f := reflect.ValueOf(value)

	// a map isn't set itself, the value is added to it
	if field.Kind() != reflect.Map && !field.CanSet() {
		return nil, f, fmt.Errorf("field isn't addressable: %v", field.Type())
	}

	if e.Kind() == reflect.Interface {
		if f.Type().Implements(e) {
			set(field, f)
			return value, f, nil
		} else if txt, ok := hookInput(f, e, hooks); ok {
			val, err := parseInterface(txt, e, hooks)
			if err != nil {
				return nil, f, err
			}

			f := reflect.ValueOf(val)
			set(field, f)
			return val, f, nil
		} else {
			return nil, f, fmt.Errorf("%v doesn't implement %v", f.Type(), e)
		}
	} else if f.Kind() == reflect.String {
		val, err := Parse(f.String(), e, hooks...)
		if err != nil {
			return nil, f, err
		}

		f := reflect.ValueOf(val)
		set(field, f)
		return val, f, nil
	} else {
		if f.Type() != e {
			if !canConvert(f, e) {
				return nil, f, fmt.Errorf("invalid data type %v for %v field", f.Type(), e)
			}

			f = f.Convert(e)
			value = f.Interface()
		}

		set(field, f)
		return value, f, nil
	}
}

// newElem returns the default of an element of type e, which isn't added to
// anything yet. An interface has no default, so its element is the zero value.
func newElem(e reflect.Type) (any, reflect.Value, error) {
	tmp, err := utils.NewWithDefaultsOf(e)
	if err != nil {
		return nil, reflect.Value{}, err
	}

	elem := reflect.ValueOf(tmp)
	if !elem.IsValid() {
		elem = reflect.Zero(e)
	}

	return tmp, elem, nil
}

// canConvert is reflect's CanConvert without the integer to string conversion,
// which would turn a number into the character with that code.
func canConvert(f reflect.Value, t reflect.Type) bool {
	if t.Kind() == reflect.String {
		switch f.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return false
		}
	}

	return f.CanConvert(t)
}

func split(path string) []string {
	quoted := false
	braces := false

	parts := strings.FieldsFunc(path, func(r rune) bool {
		if r == '"' || r == '\'' || r == '`' {
			quoted = !quoted
		}

		if r == '[' {
			braces = true
		}

		if r == ']' {
			braces = false
		}

		return !braces && !quoted && r == '.'
	})

	parts = slice_utils.Convert(parts, func(val string) string {
		return strings.Trim(val, "\"'` \t\n\r")
	})

	return parts
}
