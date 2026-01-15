// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package lookup_test

import (
	"encoding"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tiendc/go-deepcopy"
	"github.com/zauberhaus/lookup"
	"github.com/zauberhaus/random/pkg/stringer"
	utils "github.com/zauberhaus/reflect_utils"
	"go.yaml.in/yaml/v3"
)

func Ptr[T any](v T) *T {
	return &v
}

type unmarshaler struct {
	Value int
}

func (e *unmarshaler) UnmarshalText(text []byte) error {
	v, err := strconv.ParseInt(string(text), 10, 64)
	if err != nil {
		return err
	}

	e.Value = int(v)

	return nil
}

type yamlUnmarshaler struct {
	Value int
}

func (e *yamlUnmarshaler) UnmarshalYAML(v *yaml.Node) error {
	if v.Kind == yaml.ScalarNode {
		v, err := strconv.ParseInt(v.Value, 10, 64)
		if err != nil {
			return err
		}

		e.Value = int(v)
	}

	return nil
}

func cpy(val any) (any, error) {
	f := reflect.ValueOf(val)

	isPointer := f.Kind() == reflect.Pointer
	if isPointer {
		f = f.Elem()
	}

	result := reflect.New(f.Type()).Interface()

	err := deepcopy.Copy(result, val)
	if err != nil {
		return nil, err
	}

	if isPointer {
		return utils.CopyToHeap(result), nil
	}

	return result, nil
}

func Test_Copy(t *testing.T) {
	type obj[T any] struct {
		Value T
	}

	src := &obj[[]obj[int]]{[]obj[int]{{1}, {2}, {3}}}

	dst, err := cpy(src)
	assert.NoError(t, err)
	assert.Equal(t, src, dst)

	src.Value[0].Value = 99
	assert.NotEqual(t, src, dst)
}

func TestSetInterface(t *testing.T) {
	type obj struct {
		Parser encoding.TextUnmarshaler
	}

	tmp := &obj{}

	un := &unmarshaler{}
	var _ encoding.TextUnmarshaler = un

	path := "Parser"

	val, err := lookup.Get(tmp, path)
	assert.NoError(t, err)
	assert.Equal(t, nil, val)

	val, err = lookup.Set(tmp, "Parser", un)
	assert.NoError(t, err)
	assert.Equal(t, un, val)

	val, err = lookup.Get(tmp, path)
	assert.NoError(t, err)
	assert.Equal(t, un, val)

}

func TestYamlUnmarshal(t *testing.T) {
	type obj struct {
		Custom yamlUnmarshaler
	}

	tmp := &obj{}

	var zero yamlUnmarshaler
	un := yamlUnmarshaler{123}
	var _ yaml.Unmarshaler = &tmp.Custom

	path := "Custom"

	val, err := lookup.Get(tmp, path)
	assert.NoError(t, err)
	assert.Equal(t, zero, val)

	val, err = lookup.Set(tmp, path, "123")
	assert.NoError(t, err)
	assert.Equal(t, un, val)

	val, err = lookup.Get(tmp, path)
	assert.NoError(t, err)
	assert.Equal(t, un, val)
}

func TestYamlUnmarshalPointer(t *testing.T) {
	type obj struct {
		Custom *yamlUnmarshaler
	}

	tmp := &obj{}

	var zero *yamlUnmarshaler
	un := &yamlUnmarshaler{123}
	var _ yaml.Unmarshaler = tmp.Custom

	path := "Custom"

	val, err := lookup.Get(tmp, path)
	assert.NoError(t, err)
	assert.Equal(t, zero, val)

	val, err = lookup.Set(tmp, path, "123")
	assert.NoError(t, err)
	assert.Equal(t, un, val)

	val, err = lookup.Get(tmp, path)
	assert.NoError(t, err)
	assert.Equal(t, un, val)
}

func TestTextUnmarshal(t *testing.T) {
	type obj struct {
		Custom unmarshaler
	}

	tmp := &obj{}

	var zero unmarshaler
	un := unmarshaler{123}
	var _ encoding.TextUnmarshaler = &tmp.Custom

	path := "Custom"

	val, err := lookup.Get(tmp, path)
	assert.NoError(t, err)
	assert.Equal(t, zero, val)

	val, err = lookup.Set(tmp, path, "123")
	assert.NoError(t, err)
	assert.Equal(t, un, val)

	val, err = lookup.Get(tmp, path)
	assert.NoError(t, err)
	assert.Equal(t, un, val)
}

func TestTextUnmarshalPointer(t *testing.T) {
	type obj struct {
		Custom *unmarshaler
	}

	tmp := &obj{}

	var zero *unmarshaler
	un := &unmarshaler{123}
	var _ encoding.TextUnmarshaler = tmp.Custom

	path := "Custom"

	val, err := lookup.Get(tmp, path)
	assert.NoError(t, err)
	assert.Equal(t, zero, val)

	val, err = lookup.Set(tmp, path, "123")
	assert.NoError(t, err)
	assert.Equal(t, un, val)

	val, err = lookup.Get(tmp, path)
	assert.NoError(t, err)
	assert.Equal(t, un, val)
}

func TestSet_InterfaceErrors(t *testing.T) {
	type StringerContainer struct {
		Stringer fmt.Stringer
	}

	t.Run("set non-implementing type to interface field", func(t *testing.T) {
		obj := &StringerContainer{}
		// int does not implement fmt.Stringer
		_, err := lookup.Set(obj, "Stringer", 123)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "doesn't implement")
	})

	t.Run("set non-implementing type to interface in slice", func(t *testing.T) {
		type StringerSliceContainer struct {
			Stringers []fmt.Stringer
		}
		obj := &StringerSliceContainer{Stringers: []fmt.Stringer{time.Second}}
		_, err := lookup.Set(obj, "Stringers[0]", 123)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "doesn't implement")
		}
	})
}

func TestSet_InvalidType(t *testing.T) {
	type obj struct {
		Value int
	}
	v := &obj{}

	_, err := lookup.Set(v, "Value", "invalid string")
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "invalid string")
	}

	_, err = lookup.Set(v, "Value", 1.23) // float64 into int
	assert.NoError(t, err)

	_, err = lookup.Set(v, "Value", true) // bool into int
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "invalid data type")
	}
}

func Test_Get_Sub(t *testing.T) {
	type test struct {
		name     string
		obj      any
		path     string
		expected any
	}

	type obj[T any] struct {
		Value T
	}

	type sub2 struct {
		Text    string `default:"abc"`
		Integer int    `default:"100"`
	}

	type sub struct {
		Text    string `default:"xyz"`
		Integer int    `default:"99"`
		Sub     *sub2
		Slice   []string          `default:"[\"0\", \"1\", \"2\"]"`
		Map     map[string]string `default:"{\"a\": \"0\", \"b\": \"1\", \"c\": \"2\"}"`
		Map2    map[int]string
		Map3    map[float64]string
		IP      net.IP
		IPList  []net.IP
		MAC     net.HardwareAddr
		Net     net.IPNet
	}

	tests := []test{
		{
			"string",
			&obj[*sub]{},
			"value.text",
			"xyz",
		},
		{
			"integer",
			&obj[*sub]{},
			"value.integer",
			99,
		},
		{
			"slice",
			&obj[*sub]{},
			"value.slice[1]",
			"1",
		},
		{
			"string map",
			&obj[*sub]{},
			"value.map[b]",
			"1",
		},
		{
			"int map",
			&obj[*sub]{Value: &sub{Map2: map[int]string{0: "0", 1: "1", 2: "2"}}},
			"value.map2[1]",
			"1",
		},
		{
			"nil map",
			&obj[*sub]{},
			"value.map2[1]",
			"",
		},
		{
			"struct",
			&obj[*sub]{},
			"value.sub",
			(*sub2)(nil),
		},
		{
			"default text in sub struct",
			&obj[*sub]{},
			"value.sub.text",
			"abc",
		},
		{
			"not a map",
			&obj[*sub]{},
			"value.slice[a]",
			lookup.ErrNotMap,
		},
		{
			"not a map or slice",
			&obj[*sub]{},
			"value.text[1]",
			lookup.ErrNotMap,
		},
		{
			"not found",
			&obj[*sub]{},
			"value.unknown",
			errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp, err := lookup.Get(tt.obj, tt.path)
			if e, ok := tt.expected.(error); ok {
				assert.ErrorContains(t, err, e.Error())
			} else {
				if assert.NoError(t, err) {
					assert.Equal(t, tt.expected, tmp)
				}
			}
		})
	}

}

func Test_Get_Set_Errors(t *testing.T) {
	type sub struct {
		Array [1]string
		Map   map[float64]string
		Text  string
	}
	type obj[T any] struct {
		Value T
	}

	testCases := []struct {
		name           string
		obj            any
		path           string
		value          any
		expectedGet    error
		expectedSet    error
		expectedExists error
		expectedCreate error
	}{
		{
			name:           "array index out of bounds",
			obj:            &obj[sub]{},
			path:           "value.array[1]",
			expectedGet:    errors.New("array isn't expandable"),
			expectedSet:    errors.New("array isn't expandable"),
			expectedExists: errors.New("array isn't expandable"),
			expectedCreate: errors.New("array isn't expandable"),
		},
		{
			name:           "unsupported map key type for key access",
			obj:            &obj[sub]{Value: sub{Map: map[float64]string{1.0: "one"}}},
			path:           "value.map[key]",
			expectedGet:    errors.New("invalid syntax"),
			expectedSet:    errors.New("invalid syntax"),
			expectedExists: errors.New("invalid syntax"),
			expectedCreate: errors.New("invalid syntax"),
		},
		{
			name:           "not addressable field set",
			obj:            obj[int]{}, // not a pointer, so fields are not settable
			path:           "value",
			value:          "1",
			expectedSet:    errors.New("set supports only struct pointers"),
			expectedCreate: errors.New("create supports only struct pointers"),
		},
		{
			name:           "only struct",
			obj:            0,
			path:           "value",
			value:          "1",
			expectedSet:    errors.New("set supports only structs"),
			expectedGet:    errors.New("get supports only structs"),
			expectedExists: errors.New("exists supports only structs"),
			expectedCreate: errors.New("create supports only structs"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			// Test Create
			_, err := lookup.Create(tc.obj, tc.path)
			if tc.expectedCreate != nil {
				assert.ErrorContains(t, err, tc.expectedCreate.Error())
			} else {
				assert.NoError(t, err)
			}

			// Test Set
			_, err = lookup.Set(tc.obj, tc.path, tc.value)
			if tc.expectedSet != nil {
				assert.ErrorContains(t, err, tc.expectedSet.Error())
			} else {
				assert.NoError(t, err)
			}

			// Test Exists
			_, err = lookup.Exists(tc.obj, tc.path)
			if tc.expectedExists != nil {
				assert.ErrorContains(t, err, tc.expectedExists.Error())
			} else {
				assert.NoError(t, err)
			}

			// Test Get
			_, err = lookup.Get(tc.obj, tc.path)
			if tc.expectedGet != nil {
				assert.ErrorContains(t, err, tc.expectedGet.Error())
			} else {
				assert.NoError(t, err)
			}

		})
	}
}

func Test_Get_Set_Quoted(t *testing.T) {

	type obj[T any] struct {
		Value T
	}

	for _, path := range []string{"\"value[0]\"", "value['0']", "value[`0`]", "value[\"0\"]"} {

		t.Run(path, func(t *testing.T) {
			var o obj[map[int]int]

			tmp, err := lookup.Create(&o, path)
			if assert.NoError(t, err) {
				assert.Equal(t, 0, tmp)
			}

			tmp, err = lookup.Set(&o, path, "99")
			if assert.NoError(t, err) {
				assert.Equal(t, 99, tmp)
			}

			found, err := lookup.Exists(&o, path)
			if assert.NoError(t, err) {
				assert.True(t, found)
			}

			tmp, err = lookup.Get(&o, path)
			if assert.NoError(t, err) {
				assert.Equal(t, 99, tmp)
			}

		})
	}
}

func Test_Get_Set(t *testing.T) {
	type test struct {
		name  string
		value any
		obj   [2]any
	}

	type obj[T any] struct {
		Value T
	}

	type sub struct {
		Value int `default:"99"`
	}

	type StringType string
	type IntType int
	type BoolType bool
	type MapType map[string]string
	type SliceType []int
	type customDuration time.Duration

	tests := []test{
		{
			"string",
			"xyz",
			[2]any{
				&obj[string]{"xyz"},
				&obj[*string]{Ptr("xyz")},
			},
		},
		{
			"bool",
			true,
			[2]any{
				&obj[bool]{false},
				&obj[*bool]{Ptr(false)},
			},
		},
		{
			"uint8",
			uint8(12),
			[2]any{
				&obj[uint8]{12},
				&obj[*uint8]{Ptr(uint8(12))},
			},
		},
		{
			"int8",
			int8(12),
			[2]any{
				&obj[int8]{12},
				&obj[*int8]{Ptr(int8(12))},
			},
		},
		{
			"uint16",
			uint16(12),
			[2]any{
				&obj[uint16]{12},
				&obj[*uint16]{Ptr(uint16(12))},
			},
		},
		{
			"int16",
			int16(12),
			[2]any{
				&obj[int16]{12},
				&obj[*int16]{Ptr(int16(12))},
			},
		},
		{
			"uint32",
			uint32(12),
			[2]any{
				&obj[uint32]{12},
				&obj[*uint32]{Ptr(uint32(12))},
			},
		},
		{
			"int32",
			int32(12),
			[2]any{
				&obj[int32]{12},
				&obj[*int32]{Ptr(int32(12))},
			},
		},
		{
			"uint64",
			uint64(12),
			[2]any{
				&obj[uint64]{12},
				&obj[*uint64]{Ptr(uint64(12))},
			},
		},
		{
			"int64",
			int64(12),
			[2]any{
				&obj[int64]{12},
				&obj[*int64]{Ptr(int64(12))},
			},
		},
		{
			"uint",
			uint(73),
			[2]any{
				&obj[uint]{12},
				&obj[*uint]{Ptr(uint(12))},
			},
		},
		{
			"int",
			int(99),
			[2]any{
				&obj[int]{12},
				&obj[*int]{Ptr(int(12))},
			},
		},
		{
			"float32",
			float32(12.1),
			[2]any{
				&obj[float32]{12},
				&obj[*float32]{Ptr(float32(12))},
			},
		},
		{
			"float64",
			12.1,
			[2]any{
				&obj[float64]{12.1},
				&obj[*float64]{Ptr(float64(12.2))},
			},
		},
		{
			"time",
			time.Now().Round(1 * time.Second),
			[2]any{
				&obj[time.Time]{},
				&obj[*time.Time]{},
			},
		},
		{
			"duration",
			10 * time.Minute,
			[2]any{
				&obj[time.Duration]{1 * time.Minute},
				&obj[*time.Duration]{Ptr(1 * time.Minute)},
			},
		},
		{
			"decimal",
			decimal.NewFromFloat(7.7824197416),
			[2]any{
				&obj[decimal.Decimal]{},
				&obj[*decimal.Decimal]{},
			},
		},
		{
			"struct",
			sub{100},
			[2]any{
				&obj[sub]{},
				&obj[*sub]{},
			},
		},
		{
			"array",
			[]int{1, 2, 3},
			[2]any{
				&obj[[2]int]{[2]int{1, 2}},
				&obj[*[2]int]{Ptr([2]int{1, 2})},
			},
		},
		{
			"slice",
			[]int{1, 2, 3},
			[2]any{
				&obj[[]int]{},
				&obj[*[]int]{},
			},
		},
		{
			"map[string]float64",
			map[string]float64{"1": 1.1, "2": 2.2, "3": 3.3},
			[2]any{
				&obj[map[string]float64]{},
				&obj[*map[string]float64]{},
			},
		},
		{
			"map[int]float64",
			map[int]float64{1: 1.1, 2: 2.2, 3: 3.3},
			[2]any{
				&obj[map[int]float64]{},
				&obj[*map[int]float64]{},
			},
		},
		{
			"enum",
			Test3,
			[2]any{
				&obj[MyEnumer]{Test2},
				&obj[*MyEnumer]{Ptr(Test2)},
			},
		},
		{
			"net.IP",
			net.IP{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff, 0xc0, 0xa8, 0x1, 0x1},
			[2]any{
				&obj[net.IP]{net.IP{1, 1, 1, 1}},
				&obj[*net.IP]{Ptr(net.IP{1, 1, 1, 1})},
			},
		},
		{
			"net.HardwareAddr",
			net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
			[2]any{
				&obj[net.HardwareAddr]{net.HardwareAddr{1, 1, 1, 1, 1, 1}},
				&obj[*net.HardwareAddr]{Ptr(net.HardwareAddr{1, 1, 1, 1, 1, 1})},
			},
		},
		{
			"net.IPNet",
			net.IPNet{IP: net.IP{0xc0, 0xa8, 0x1, 0x0}, Mask: net.IPMask{0xff, 0xff, 0xff, 0x0}},
			[2]any{
				&obj[net.IPNet]{net.IPNet{IP: net.IP{1, 1, 1, 1}, Mask: net.IPMask{0xff, 0xff, 0x00, 0x0}}},
				&obj[*net.IPNet]{Ptr(net.IPNet{IP: net.IP{1, 1, 1, 1}, Mask: net.IPMask{0xff, 0xff, 0x00, 0x0}})},
			},
		},
		{
			"fmt.Stringer",
			10 * time.Second,
			[2]any{
				&obj[fmt.Stringer]{},
				&obj[*fmt.Stringer]{},
			},
		},
		{
			"string type",
			StringType("abc"),
			[2]any{
				&obj[StringType]{},
				&obj[*StringType]{},
			},
		},
		{
			"int type",
			IntType(999),
			[2]any{
				&obj[IntType]{},
				&obj[*IntType]{},
			},
		},
		{
			"bool type",
			BoolType(true),
			[2]any{
				&obj[BoolType]{},
				&obj[*BoolType]{},
			},
		},
		{
			"map type",
			MapType(map[string]string{"a x": "b y", "d": "e"}),
			[2]any{
				&obj[MapType]{},
				&obj[*MapType]{},
			},
		},
		{
			"slice type",
			SliceType([]int{1, 2, 3, 4, 5}),
			[2]any{
				&obj[SliceType]{},
				&obj[*SliceType]{},
			},
		},
		{
			"custom duration type",
			customDuration(999 * time.Second),
			[2]any{
				&obj[customDuration]{},
				&obj[*customDuration]{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tests := []string{"value", "pointer", "to pointer", "from pointer", "parse", "parse to pointer", "zero"}

			if tt.name == "fmt.Stringer" {
				tests = []string{"value", "zero"}
			}

			for _, test := range tests {

				t.Run(test, func(t *testing.T) {
					obj, err := cpy(tt.obj)
					require.NoError(t, err)

					value := tt.value

					var expected any

					switch test {
					case "value":
						obj = tt.obj[0]
					case "pointer":
						value = utils.CopyToHeap(value)
						obj = tt.obj[1]
					case "from pointer":
						value = utils.CopyToHeap(value)
						obj = tt.obj[0]
					case "to pointer":
						obj = tt.obj[1]
					case "parse", "parse to pointer":
						expected = value
						value, err = stringer.String(value)
						assert.NoError(t, err)

						if test == "parse to pointer" {
							obj = tt.obj[1]
						} else {
							obj = tt.obj[0]
						}

					case "zero":
						obj = tt.obj[0]
						value = nil
					}

					o := reflect.ValueOf(obj)
					v := reflect.ValueOf(value)

					isPtr := v.Kind() == reflect.Pointer

					if expected != nil {
						isPtr = reflect.TypeOf(expected).Kind() == reflect.Pointer
					}

					if o.Kind() == reflect.Pointer {
						o = o.Elem()
					}

					f := o.Field(0)
					isPtr2 := f.Kind() == reflect.Pointer

					orig := f.Interface()

					if expected == nil {
						expected = value
						if utils.IsNil(value) {
							tp := f.Type()
							z := reflect.Zero(tp)
							expected = z.Interface()
						}
					}

					if !isPtr && isPtr2 {
						expected = utils.CopyToHeap(expected)
					} else if isPtr && !isPtr2 {
						expected = utils.FromPointer(expected)
					}

					tmp, err := lookup.Get(obj, "value")
					if assert.NoError(t, err) {
						assert.Equal(t, orig, tmp)
					}

					if f.Kind() == reflect.Array {
						return
					}

					if f.Kind() == reflect.Pointer && f.Elem().Kind() == reflect.Array {
						return
					}

					tmp, err = lookup.Set(obj, "value", value)
					if assert.NoError(t, err) {
						assert.Equal(t, expected, tmp)
					}

					tmp, err = lookup.Get(obj, "value")
					if assert.NoError(t, err) {
						assert.Equal(t, expected, tmp)
					}

				})
			}
		})
	}
}

func Test_Get_Set_Slice(t *testing.T) {
	type test struct {
		name  string
		value any
		obj   any
		index int
	}

	type obj[T any] struct {
		Value T
	}

	tests := []test{
		{
			"int slice",
			99,
			&obj[[]int]{[]int{0, 1, 2}},
			1,
		},
		{
			"sub slice",
			[]string{"abc", "xyz"},
			&obj[[][]string]{[][]string{{"0"}, {"1"}, {"2"}}},
			1,
		},
		{
			"map slice",
			map[string]string{"1": "abc", "0": "xyz"},
			&obj[[]map[string]string]{[]map[string]string{{"a": "0"}, {"b": "1"}, {"c": "2"}}},
			1,
		},
		{
			"extend sub slice",
			[]string{"abc", "xyz"},
			&obj[[][]string]{[][]string{{"0"}, {"1"}, {"2"}}},
			5,
		},
		{
			"struct slice",
			obj[int]{4},
			&obj[[]obj[int]]{[]obj[int]{{0}, {1}, {2}}},
			1,
		},
		{
			"extend struct slice",
			obj[int]{4},
			&obj[[]obj[int]]{[]obj[int]{{0}, {1}, {2}}},
			5,
		},
		{
			"string array",
			"xyz",
			&obj[[3]string]{[3]string{"0", "1", "2"}},
			1,
		},
		{
			"string slice",
			"xyz",
			&obj[[]string]{[]string{"0", "1", "2"}},
			1,
		},
		{
			"extend string slice",
			"xyz",
			&obj[[]string]{[]string{"0", "1", "2"}},
			3,
		},
		{
			"string pointer slice",
			Ptr("xyz"),
			&obj[[]*string]{[]*string{Ptr("000"), Ptr("001"), Ptr("002")}},
			1,
		},
		{
			"extend string pointer slice",
			Ptr("xyz"),
			&obj[[]*string]{[]*string{Ptr("000"), Ptr("001"), Ptr("002")}},
			3,
		},
		{
			"replace",
			[]*string{Ptr("003"), Ptr("004"), Ptr("005")},
			&obj[[]*string]{[]*string{Ptr("000"), Ptr("001"), Ptr("002")}},
			-1,
		},
		{
			"append",
			3,
			&obj[[]int]{[]int{0, 1, 2}},
			-2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := "value"

			if tt.index >= 0 {
				path = fmt.Sprintf("value[%d]", tt.index)
			} else if tt.index == -2 {
				path = "value[]"
			}

			d := reflect.ValueOf(tt.obj)

			if utils.IsPointer(d) {
				d = d.Elem()
			}

			f := d.Field(0)

			l := f.Len()

			tmp, err := lookup.Set(tt.obj, path, tt.value)
			if assert.NoError(t, err) {
				if assert.NotNil(t, tmp) {
					assert.Equal(t, tt.value, tmp)
				}
			}

			if tt.index != -2 {

				tmp, err = lookup.Get(tt.obj, path)
				if assert.NoError(t, err) {
					if assert.NotNil(t, tmp) {
						assert.Equal(t, tt.value, tmp)
					}
				}

				if tt.index+1 > l {
					l = tt.index + 1
				}

				l2 := f.Len()
				assert.Equal(t, l, l2)

			} else {
				path = fmt.Sprintf("value[%d]", l)
				tmp, err = lookup.Get(tt.obj, path)
				if assert.NoError(t, err) {
					if assert.NotNil(t, tmp) {
						assert.Equal(t, tt.value, tmp)
					}
				}

				l2 := f.Len()
				assert.Equal(t, l+1, l2)
			}

		})
	}

}

func Test_Set_Array_Sub(t *testing.T) {
	type test struct {
		name  string
		path  string
		value any
		obj   any
		index int
	}

	type obj[T any] struct {
		Value T
	}

	tests := []test{
		{
			"sub slice",
			"",
			[]string{"xyz"},
			&obj[[][]string]{[][]string{{"0"}, {"1"}, {"2"}}},
			1,
		},
		{
			"string slice",
			"",
			"xyz",
			&obj[[]string]{[]string{"0", "1", "2"}},
			1,
		},
		{
			"extend string slice",
			"",
			"xyz",
			&obj[[]string]{[]string{"0", "1", "2"}},
			3,
		},
		{
			"string pointer slice",
			"",
			Ptr("xyz"),
			&obj[[]*string]{[]*string{Ptr("000"), Ptr("001"), Ptr("002")}},
			1,
		},
		{
			"extend string pointer slice",
			"",
			Ptr("xyz"),
			&obj[[]*string]{[]*string{Ptr("000"), Ptr("001"), Ptr("002")}},
			3,
		},
		{
			"sub array",
			"value[1].value",
			99,
			&obj[[]obj[int]]{[]obj[int]{{1}, {2}, {3}}},
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, parsed := range []bool{true, false} {
				name := "native"
				if parsed {
					name = "parsed"
				}

				t.Run(name, func(t *testing.T) {
					path := tt.path

					if len(path) == 0 {
						path = fmt.Sprintf("value[%d]", tt.index)
					}

					expected := tt.value

					value := tt.value
					if parsed {
						if utils.IsPointer(value) {
							value = utils.FromPointer(value)
						}

						switch o := value.(type) {
						case []string:
							value = strings.Join(o, ",")
						default:
							value = fmt.Sprintf("%v", value)
						}

					}

					obj, err := cpy(tt.obj)
					require.NoError(t, err)

					d := reflect.ValueOf(obj)

					if utils.IsPointer(d) {
						d = d.Elem()
					}

					f := d.Field(0)

					l := f.Len()

					tmp, err := lookup.Set(obj, path, value)
					if assert.NoError(t, err) {
						if assert.NotNil(t, tmp) {
							assert.Equal(t, expected, tmp)
						}
					}

					tmp, err = lookup.Get(obj, path)
					if assert.NoError(t, err) {
						if assert.NotNil(t, tmp) {
							assert.Equal(t, expected, tmp)
						}
					}

					if tt.index+1 > l {
						l = tt.index + 1
					}

					l2 := f.Len()
					assert.Equal(t, l, l2)

				})
			}
		})
	}
}

func Test_Set_Parse_Error(t *testing.T) {
	type test struct {
		name        string
		path        string
		value       string
		obj         any
		expectedErr string // Substring of the expected error message
	}

	type obj[T any] struct {
		Value T
	}

	tests := []test{
		// Invalid boolean string
		{
			"invalid bool string",
			"value",
			"not-a-bool",
			&obj[bool]{},
			"strconv.ParseBool: parsing \"not-a-bool\": invalid syntax",
		},
		// Invalid integer string
		{
			"invalid int string",
			"value",
			"abc",
			&obj[int]{},
			"strconv.ParseInt: parsing \"abc\": invalid syntax",
		},
		// Integer overflow
		{
			"int overflow",
			"value",
			"9223372036854775808", // Max int64 + 1
			&obj[int64]{},
			"strconv.ParseInt: parsing \"9223372036854775808\": value out of range",
		},
		// Invalid float string
		{
			"invalid float string",
			"value",
			"xyz",
			&obj[float64]{},
			"strconv.ParseFloat: parsing \"xyz\": invalid syntax",
		},
		// Invalid duration string
		{
			"invalid duration string",
			"value",
			"10minutes", // Should be "10m"
			&obj[time.Duration]{},
			"time: unknown unit \"minutes\" in duration \"10minutes\"",
		},
		// Invalid enum string
		{
			"invalid enum string",
			"value",
			"UnknownPolicy",
			&obj[MyEnumer]{},
			"UnknownPolicy does not belong to MyEnumer values",
		},
		// Slice of ints with invalid element
		{
			"slice of ints with invalid element",
			"value",
			"1,two,3",
			&obj[[]int]{},
			"strconv.ParseInt: parsing \"two\": invalid syntax",
		},
		{
			"map string to int with invalid value",
			"value",
			"a=1,b=two",
			&obj[map[string]int]{},
			"strconv.ParseInt: parsing \"two\": invalid syntax",
		},
		{
			"map string to int with invalid key",
			"value",
			"1=1,b=2",
			&obj[map[int]int]{},
			"strconv.ParseInt: parsing \"b\": invalid syntax",
		},
		{
			"struct from malformed json",
			"value",
			`{"Value": }`,
			&obj[obj[int]]{},
			"invalid character '}' looking for beginning of value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, err := cpy(tt.obj) // Use the existing copy helper
			require.NoError(t, err)

			_, err = lookup.Set(obj, tt.path, tt.value)
			assert.ErrorContains(t, err, tt.expectedErr)
		})
	}
}

func Test_Get_Array(t *testing.T) {
	type test struct {
		name     string
		pos      int
		obj      any
		expected any
	}

	type obj[T any] struct {
		Value T
	}

	type obj2 struct {
		Value string `default:"test value"`
	}

	tests := []test{
		{
			"struct array",
			2,
			&obj[[]obj[int]]{[]obj[int]{{0}, {1}, {2}}},
			obj[int]{2},
		},
		{
			"struct pointer array",
			2,
			&obj[[]*obj[int]]{[]*obj[int]{{0}, {1}, {2}}},
			Ptr(obj[int]{2}),
		},
		{
			"string array",
			2,
			&obj[[3]string]{[3]string{"000", "001", "002"}},
			"002",
		},
		{
			"string slice",
			1,
			&obj[[]string]{[]string{"000", "001", "002"}},
			"001",
		},
		{
			"extend string slice",
			3,
			&obj[[]string]{[]string{"000", "001", "002"}},
			"",
		},
		{
			"string pointer slice",
			1,
			&obj[[]*string]{[]*string{Ptr("000"), Ptr("001"), Ptr("002")}},
			Ptr("001"),
		},
		{
			"extend string pointer slice",
			3,
			&obj[[]*string]{[]*string{Ptr("000"), Ptr("001"), Ptr("002")}},
			Ptr(""),
		},
		{
			"extend struct array",
			3,
			&obj[[]obj2]{[]obj2{{"0"}, {"1"}, {"2"}}},
			obj2{"test value"},
		},
		{
			"extend struct pointer array",
			3,
			&obj[[]*obj2]{[]*obj2{{"0"}, {"1"}, {"2"}}},
			Ptr(obj2{"test value"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := reflect.ValueOf(tt.obj)

			if utils.IsPointer(d) {
				d = d.Elem()
			}

			f := d.Field(0)

			l := f.Len()

			if tt.expected == nil {

				if l > tt.pos {
					e := f.Index(tt.pos)

					assert.True(t, e.CanSet())
					assert.True(t, e.IsValid())
					assert.True(t, e.CanInterface())

					var err error
					tt.expected = e.Interface()
					if utils.IsNil(tt.expected) {
						tt.expected, err = utils.NewWithDefaultsOf(e.Type())
					}

					assert.NoError(t, err)
				} else {
					var err error
					tt.expected, err = utils.NewWithDefaultsOf(f.Type().Elem())
					assert.NoError(t, err)
				}
			}

			path := fmt.Sprintf("value[%d]", tt.pos)
			tmp, err := lookup.Get(tt.obj, path)
			if assert.NoError(t, err) {
				if assert.NotNil(t, tmp) {
					assert.Equal(t, tt.expected, tmp)
				}
			}

			if tt.pos+1 > l {
				l = tt.pos + 1
			}

			l2 := f.Len()
			assert.Equal(t, l, l2)

		})
	}

}

func Test_Get_Set_Map(t *testing.T) {
	type test struct {
		name  string
		obj   any
		key   string
		value any
	}

	type obj[T any] struct {
		Value T
	}

	tests := []test{
		{
			"new entry",
			&obj[map[string]int]{map[string]int{"key": 1}},
			"new",
			0,
		},
		{
			"int",
			&obj[map[string]int]{map[string]int{"key": 1}},
			"key",
			1,
		},
		{
			"int pointer",
			&obj[map[string]*int]{map[string]*int{"key": Ptr(1)}},
			"key",
			Ptr(1),
		},
		{
			"new int pointer",
			&obj[map[string]*int]{map[string]*int{"key": Ptr(1)}},
			"new",
			Ptr(0),
		},
		{
			"remove entry",
			&obj[map[string]*int]{map[string]*int{"key": Ptr(1)}},
			"key",
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			obj, err := cpy(tt.obj)
			require.NoError(t, err)

			o := reflect.ValueOf(obj)
			f := o.Elem().Field(0)

			k := reflect.ValueOf(tt.key)
			v := f.MapIndex(k)

			path := fmt.Sprintf("value[%s]", tt.key)

			if (v != reflect.Value{}) {
				var expected any
				if v.CanInterface() {
					expected = v.Interface()
				}

				tmp, err := lookup.Get(obj, path)
				if assert.NoError(t, err) {
					assert.Equal(t, expected, tmp)
				}
			}

			tmp, err := lookup.Set(obj, path, tt.value)
			if assert.NoError(t, err) {
				assert.Equal(t, tt.value, tmp)
			}

			if tt.value != nil {
				tmp, err = lookup.Get(obj, path)
				if assert.NoError(t, err) {
					assert.Equal(t, tt.value, tmp)
				}
			}

		})
	}

}

func Test_Get_Struct_Array_Field(t *testing.T) {
	type test struct {
		name     string
		pos      int
		obj      any
		expected any
	}

	type obj[T any] struct {
		Value T
	}

	type obj2 struct {
		Value string `default:"test value"`
	}

	tests := []test{
		{
			"struct array",
			2,
			&obj[[]obj[int]]{[]obj[int]{{0}, {1}, {2}}},
			2,
		},
		{
			"struct pointer array",
			2,
			&obj[[]*obj[int]]{[]*obj[int]{{0}, {1}, {2}}},
			2,
		},
		{
			"extend struct array",
			3,
			&obj[[]obj2]{[]obj2{{"0"}, {"1"}, {"2"}}},
			"test value",
		},
		{
			"extend struct pointer array",
			3,
			&obj[[]*obj2]{[]*obj2{{"0"}, {"1"}, {"2"}}},
			"test value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := reflect.ValueOf(tt.obj)

			if utils.IsPointer(d) {
				d = d.Elem()
			}

			f := d.Field(0)

			l := f.Len()

			if tt.expected == nil {

				if l > tt.pos {
					e := f.Index(tt.pos)

					assert.True(t, e.CanSet())
					assert.True(t, e.IsValid())
					assert.True(t, e.CanInterface())

					var err error
					tt.expected = e.Interface()
					if utils.IsNil(tt.expected) {
						tt.expected, err = utils.NewWithDefaultsOf(e.Type())
					}

					assert.NoError(t, err)
				} else {
					var err error
					tt.expected, err = utils.NewWithDefaultsOf(f.Type().Elem())
					assert.NoError(t, err)
				}
			}

			path := fmt.Sprintf("value[%d].value", tt.pos)
			tmp, err := lookup.Get(tt.obj, path)
			if assert.NoError(t, err) {
				if assert.NotNil(t, tmp) {
					assert.Equal(t, tt.expected, tmp)
				}
			}

			if tt.pos+1 > l {
				l = tt.pos + 1
			}

			l2 := f.Len()
			assert.Equal(t, l, l2)

		})
	}

}

type sampleStruct struct {
	Name            string
	Value           int
	Nested          *nestedStruct
	Items           []itemStruct
	Map             map[string]itemStruct
	PtrMap          map[string]*itemStruct
	NilMap          map[string]string
	NilSlice        []string
	unexportedField string
}

type nestedStruct struct {
	Data string
}

type itemStruct struct {
	ID int
}

func TestHas(t *testing.T) {
	sample := &sampleStruct{
		Name:  "test",
		Value: 123,
		Nested: &nestedStruct{
			Data: "nested data",
		},
		Items: []itemStruct{
			{ID: 1},
			{ID: 2},
		},
		Map: map[string]itemStruct{
			"first": {ID: 10},
		},
		PtrMap: map[string]*itemStruct{
			"one": {ID: 100},
		},
	}

	tests := []struct {
		name      string
		path      string
		wantHas   bool
		assertion assert.ErrorAssertionFunc
	}{
		{"simple field exists", "Name", true, assert.NoError},
		{"simple field exists (case-insensitive)", "name", true, assert.NoError},
		{"simple field does not exist", "NonExistent", false, assert.Error},
		{"nested field exists", "Nested.Data", true, assert.NoError},
		{"nested field does not exist", "Nested.NonExistent", false, assert.Error},
		{"item in slice exists", "Items[0]", true, assert.NoError},
		{"item in slice out of bounds", "Items[2]", false, assert.NoError},
		{"field in item in slice exists", "Items[0].ID", true, assert.NoError},
		{"field in item in slice out of bounds", "Items[2].ID", false, assert.NoError},
		{"item in map exists", `Map["first"]`, true, assert.NoError},
		{"item in map does not exist", `Map["second"]`, false, assert.NoError},
		{"field in item in map exists", `Map["first"].ID`, true, assert.NoError},
		{"field in item in map does not exist", `Map["second"].ID`, false, assert.NoError},
		{"item in pointer map exists", `PtrMap["one"]`, true, assert.NoError},
		{"field in item in pointer map exists", `PtrMap["one"].ID`, true, assert.NoError},
		{"path with single quotes", "Map['first']", true, assert.NoError},
		{"path with backticks", "Map[`first`]", true, assert.NoError},
		{"path with dot inside quotes", `Map["first.key"]`, false, assert.NoError},
		{"unexported field", "unexportedField", false, assert.Error},
		{"nil nested struct", "Nested.Data", true, assert.NoError}, // Test on a different sample
		{"nil map", "NilMap.key", false, assert.NoError},
		{"nil slice", "NilSlice[0]", false, assert.NoError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Special case for nil nested struct test
			if tt.name == "nil nested struct" {
				sampleWithNilNested := &sampleStruct{Name: "test"}
				has, err := lookup.Exists(sampleWithNilNested, "Nested.Data")
				assert.False(t, has)
				assert.NoError(t, err)
				return
			}

			has, err := lookup.Exists(sample, tt.path)
			assert.Equal(t, tt.wantHas, has)
			tt.assertion(t, err)
		})
	}

	t.Run("non-pointer struct input", func(t *testing.T) {
		has, err := lookup.Exists(*sample, "Name")
		assert.True(t, has)
		assert.NoError(t, err)
	})

	t.Run("path with spaces around dot", func(t *testing.T) {
		has, err := lookup.Exists(sample, "Nested . Data")
		assert.True(t, has)
		assert.NoError(t, err)
	})

	t.Run("empty path", func(t *testing.T) {
		has, err := lookup.Exists(sample, "")
		assert.False(t, has)
		assert.NoError(t, err)
	})

	t.Run("path is just a dot", func(t *testing.T) {
		has, err := lookup.Exists(sample, ".")
		assert.False(t, has)
		assert.NoError(t, err)
	})

	t.Run("path with multiple dots", func(t *testing.T) {
		has, err := lookup.Exists(sample, "Nested..Data")
		assert.True(t, has)
		assert.NoError(t, err)
	})

	t.Run("path with trailing dot", func(t *testing.T) {
		has, err := lookup.Exists(sample, "Nested.")
		assert.True(t, has)
		assert.NoError(t, err)
	})

	t.Run("path with leading dot", func(t *testing.T) {
		has, err := lookup.Exists(sample, ".Nested")
		assert.True(t, has)
		assert.NoError(t, err)
	})

	t.Run("has on nil map returns false", func(t *testing.T) {
		s := &sampleStruct{}
		has, err := lookup.Exists(s, "NilMap.key")
		assert.False(t, has)
		assert.NoError(t, err)
	})

	t.Run("has on nil slice returns false", func(t *testing.T) {
		s := &sampleStruct{}
		has, err := lookup.Exists(s, "NilSlice[0]")
		assert.False(t, has)
		assert.NoError(t, err)
	})
}
