// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package lookup

import (
	"encoding"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/creasty/defaults"
	utils "github.com/zauberhaus/reflect_utils"
	"go.yaml.in/yaml/v3"
)

var (
	tum = reflect.TypeFor[encoding.TextUnmarshaler]()
	yum = reflect.TypeFor[yaml.Unmarshaler]()
)

func Split(txt string, sep rune) []string {
	var result []string
	var start int
	var inQuote, inSingleQuote, inBacktick, inBracket, inBrace bool

	for i, r := range txt {
		switch r {
		case '"':
			if !inSingleQuote && !inBacktick {
				inQuote = !inQuote
			}
		case '\'':
			if !inQuote && !inBacktick {
				inSingleQuote = !inSingleQuote
			}
		case '`':
			if !inQuote && !inSingleQuote {
				inBacktick = !inBacktick
			}
		case '[':
			if !inQuote && !inSingleQuote && !inBacktick {
				inBracket = true
			}
		case ']':
			if !inQuote && !inSingleQuote && !inBacktick {
				inBracket = false
			}
		case '{':
			if !inQuote && !inSingleQuote && !inBacktick {
				inBrace = true
			}
		case '}':
			if !inQuote && !inSingleQuote && !inBacktick {
				inBrace = false
			}
		case sep:
			if !inQuote && !inSingleQuote && !inBacktick && !inBracket && !inBrace {
				result = append(result, txt[start:i])
				start = i + 1
			}
		}
	}
	result = append(result, txt[start:])
	return result
}

func Parse(txt string, t reflect.Type, hooks ...ParserHook) (any, error) {
	for _, hook := range hooks {
		if hook.To == t {
			return hook.Parse(txt)
		}
	}

	switch t {
	case reflect.TypeFor[string]():
		return txt, nil

	case reflect.TypeFor[*string]():
		return &txt, nil
	}

	isPointer := t.Kind() == reflect.Pointer
	if isPointer {
		if t.Implements(tum) {
			v := reflect.New(t.Elem()).Interface()
			err := v.(encoding.TextUnmarshaler).UnmarshalText([]byte(txt))
			if err != nil {
				return nil, err
			}

			return v, nil
		} else if t.Implements(yum) {
			v := reflect.New(t.Elem()).Interface()
			n := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: txt,
			}

			err := v.(yaml.Unmarshaler).UnmarshalYAML(n)
			if err != nil {
				return nil, err
			}

			return v, nil
		}
	} else {
		v := reflect.New(t)
		if v.Type().Implements(tum) {
			val := v.Interface().(encoding.TextUnmarshaler)
			err := val.UnmarshalText([]byte(txt))
			if err != nil {
				return nil, err
			}

			return v.Elem().Interface(), nil
		} else if v.Type().Implements(yum) {
			val := v.Interface().(yaml.Unmarshaler)
			n := &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: txt,
			}

			err := val.UnmarshalYAML(n)
			if err != nil {
				return nil, err
			}

			return v.Elem().Interface(), nil
		}
	}

	if isPointer {
		t = t.Elem()
	}

	switch t {
	case reflect.TypeFor[time.Duration]():
		v, err := time.ParseDuration(txt)
		if err != nil {
			return nil, err
		}

		if isPointer {
			return &v, nil
		} else {
			return v, nil
		}
	case reflect.TypeFor[net.HardwareAddr]():
		v, err := net.ParseMAC(txt)
		if err != nil {
			return nil, err
		}

		if isPointer {
			return &v, nil
		} else if v != nil {
			return v, nil
		} else {
			return net.HardwareAddr{}, nil
		}
	case reflect.TypeFor[net.IPNet]():
		_, v, err := net.ParseCIDR(txt)
		if err != nil {
			return nil, err
		}

		if isPointer {
			return v, nil
		} else if v != nil {
			return *v, nil
		} else {
			return net.IPNet{}, nil
		}
	}

	switch t.Kind() {
	case reflect.String:
		val := reflect.ValueOf(txt)

		if val.CanConvert(t) {
			val = val.Convert(t)
		}

		if isPointer {
			return utils.CopyToHeap(val.Interface()), nil
		} else {
			return val.Interface(), nil
		}

	case reflect.Bool:
		v, err := strconv.ParseBool(txt)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Int8:
		v, err := strconv.ParseInt(txt, 10, 8)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Int16:
		v, err := strconv.ParseInt(txt, 10, 16)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Int32:
		v, err := strconv.ParseInt(txt, 10, 32)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Int64:
		v, err := strconv.ParseInt(txt, 10, 64)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Int:
		v, err := strconv.ParseInt(txt, 10, 64)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Uint:
		v, err := strconv.ParseUint(txt, 10, 64)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Uint8:
		v, err := strconv.ParseUint(txt, 10, 8)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Uint16:
		v, err := strconv.ParseUint(txt, 10, 16)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Uint32:
		v, err := strconv.ParseUint(txt, 10, 32)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Uint64:
		v, err := strconv.ParseUint(txt, 10, 64)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Float64:
		v, err := strconv.ParseFloat(txt, 64)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Float32:
		v, err := strconv.ParseFloat(txt, 32)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Complex64:
		v, err := strconv.ParseComplex(txt, 64)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Complex128:
		v, err := strconv.ParseComplex(txt, 128)
		if err != nil {
			return nil, err
		}

		return convert(v, t, isPointer)

	case reflect.Slice:
		e := t.Elem()

		if e.Kind() == reflect.Uint8 {
			if strings.HasPrefix(txt, "0x") {
				txt = strings.TrimPrefix(txt, "0x")
				data, err := hex.DecodeString(txt)
				if err != nil {
					return nil, err
				}

				if isPointer {
					return &data, nil
				} else {
					return data, nil
				}
			}
		}

		if len(txt) == 0 {
			s := reflect.MakeSlice(t, 0, 0)

			if isPointer {
				return utils.CopyToHeap(s.Interface()), nil
			} else {
				return s.Interface(), nil

			}
		}

		parts := Split(txt, ',')
		s := reflect.MakeSlice(t, 0, len(parts))

		for _, p := range parts {
			v, err := Parse(p, e)
			if err != nil {
				return nil, err
			}

			s = reflect.Append(s, reflect.ValueOf(v))
		}

		return convert(s.Interface(), t, isPointer)

	case reflect.Array:
		e := t.Elem()

		if e.Kind() == reflect.Uint8 {
			if strings.HasPrefix(txt, "0x") {
				txt = strings.TrimPrefix(txt, "0x")
				data, err := hex.DecodeString(txt)
				if err != nil {
					return nil, err
				}

				s := reflect.New(t).Elem()
				reflect.Copy(s, reflect.ValueOf(data))

				if isPointer {
					return utils.CopyToHeap(s.Interface()), nil
				} else {
					return s.Interface(), nil
				}
			}
		}

		if len(txt) == 0 {
			s := reflect.New(t).Elem()

			if isPointer {
				return utils.CopyToHeap(s.Interface()), nil
			} else {
				return s.Interface(), nil

			}
		}

		parts := Split(txt, ',')
		s := reflect.New(t).Elem()

		if len(parts) != s.Len() {
			return nil, fmt.Errorf("expected %d elements, got %d", s.Len(), len(parts))
		}

		for i, p := range parts {
			v, err := Parse(p, e)
			if err != nil {
				return nil, err
			}

			s.Index(i).Set(reflect.ValueOf(v))
		}

		return convert(s.Interface(), t, isPointer)

	case reflect.Map:

		parts := Split(txt, ',')
		s := reflect.MakeMap(t)

		kt := t.Key()
		vt := t.Elem()

		for _, p := range parts {
			kv := strings.Split(p, "=")
			if len(kv) == 2 {

				k, err := Parse(kv[0], kt)
				if err != nil {
					return nil, err
				}

				v, err := Parse(kv[1], vt)
				if err != nil {
					return nil, err
				}

				s.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
			}
		}

		return convert(s.Interface(), t, isPointer)

	case reflect.Struct:
		val, err := utils.NewOf(t)
		if err != nil {
			return nil, fmt.Errorf("new failed: %w", err)
		}

		err = defaults.Set(val)
		if err != nil {
			return nil, fmt.Errorf("set defaults failed: %w", err)
		}

		err = json.Unmarshal([]byte(txt), val)
		if err != nil {
			return nil, err
		}

		if isPointer {
			return val, nil
		} else {
			v := reflect.ValueOf(val)
			e := v.Elem()
			return e.Interface(), nil
		}
	}

	return nil, fmt.Errorf("unsupported data type: %v", t)

}

func convert(v any, t reflect.Type, isPointer bool) (any, error) {
	if t == reflect.TypeOf(v) {
		if isPointer {
			return utils.CopyToHeap(v), nil
		} else {
			return v, nil
		}
	} else {
		vt := reflect.ValueOf(v)
		if vt.Type().ConvertibleTo(t) {
			vt = vt.Convert(t)
		}

		if isPointer {
			return utils.CopyToHeap(vt.Interface()), nil
		} else {
			return vt.Interface(), nil
		}
	}
}
