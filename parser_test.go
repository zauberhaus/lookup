// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package lookup_test

import (
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zauberhaus/lookup"
)

type jsonStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type errorUnmarshaler struct{}

func (e *errorUnmarshaler) UnmarshalText(text []byte) error {
	return assert.AnError
}

func TestSplit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		input    string
		sep      rune
		expected []string
	}{
		{
			name:     "simple split",
			input:    "a,b,c",
			sep:      ',',
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "separator inside double quotes",
			input:    `a,"b,c",d`,
			sep:      ',',
			expected: []string{"a", `"b,c"`, "d"},
		},
		{
			name:     "separator inside single quotes",
			input:    `a,'b,c',d`,
			sep:      ',',
			expected: []string{"a", `'b,c'`, "d"},
		},
		{
			name:     "separator inside backticks",
			input:    "a,`b,c`,d",
			sep:      ',',
			expected: []string{"a", "`b,c`", "d"},
		},
		{
			name:     "separator inside square brackets",
			input:    "a,[b,c],d",
			sep:      ',',
			expected: []string{"a", "[b,c]", "d"},
		},
		{
			name:     "separator inside curly braces",
			input:    `a,{"b":"c"},d`,
			sep:      ',',
			expected: []string{"a", `{"b":"c"}`, "d"},
		},
		{
			name:     "mixed delimiters",
			input:    `a,"b,c",{d,e},[f,g],h`,
			sep:      ',',
			expected: []string{"a", `"b,c"`, "{d,e}", "[f,g]", "h"},
		},
		{
			name:     "consecutive separators",
			input:    "a,,b,c",
			sep:      ',',
			expected: []string{"a", "", "b", "c"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := lookup.Split(tc.input, tc.sep)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	ptr := func(v any) any {
		p := reflect.New(reflect.TypeOf(v))
		p.Elem().Set(reflect.ValueOf(v))
		return p.Interface()
	}

	type obj struct {
		Name  string
		Value float64
	}

	tests := []struct {
		name        string
		txt         string
		typ         reflect.Type
		expectedVal any
		//expectErr   string
	}{
		{
			name:        "string",
			txt:         "hello",
			typ:         reflect.TypeFor[string](),
			expectedVal: "hello",
		},
		{
			name:        "pointer to string",
			txt:         "world",
			typ:         reflect.TypeFor[*string](),
			expectedVal: ptr("world"),
		},
		{
			name:        "bool true",
			txt:         "true",
			typ:         reflect.TypeFor[bool](),
			expectedVal: true,
		},
		{
			name:        "pointer to bool false",
			txt:         "false",
			typ:         reflect.TypeFor[*bool](),
			expectedVal: ptr(false),
		},
		{
			name:        "int",
			txt:         "123",
			typ:         reflect.TypeFor[int](),
			expectedVal: 123,
		},
		{
			name:        "pointer to int",
			txt:         "-45",
			typ:         reflect.TypeFor[*int](),
			expectedVal: ptr(-45),
		},
		{
			name:        "int8",
			txt:         "127",
			typ:         reflect.TypeFor[int8](),
			expectedVal: int8(127),
		},
		{
			name:        "pointer to int8",
			txt:         "-128",
			typ:         reflect.TypeFor[*int8](),
			expectedVal: ptr(int8(-128)),
		},
		{
			name:        "int16",
			txt:         "32767",
			typ:         reflect.TypeFor[int16](),
			expectedVal: int16(32767),
		},
		{
			name:        "pointer to int16",
			txt:         "-32768",
			typ:         reflect.TypeFor[*int16](),
			expectedVal: ptr(int16(-32768)),
		},
		{
			name:        "int32",
			txt:         "2147483647",
			typ:         reflect.TypeFor[int32](),
			expectedVal: int32(2147483647),
		},
		{
			name:        "pointer to int32",
			txt:         "-2147483648",
			typ:         reflect.TypeFor[*int32](),
			expectedVal: ptr(int32(-2147483648)),
		},
		{
			name:        "uint8",
			txt:         "255",
			typ:         reflect.TypeFor[uint8](),
			expectedVal: uint8(255),
		},
		{
			name:        "pointer to uint8",
			txt:         "128",
			typ:         reflect.TypeFor[*uint8](),
			expectedVal: ptr(uint8(128)),
		},
		{
			name:        "uint16",
			txt:         "65535",
			typ:         reflect.TypeFor[uint16](),
			expectedVal: uint16(65535),
		},
		{
			name:        "pointer to uint16",
			txt:         "1000",
			typ:         reflect.TypeFor[*uint16](),
			expectedVal: ptr(uint16(1000)),
		},
		{
			name:        "uint32",
			txt:         "4294967295",
			typ:         reflect.TypeFor[uint32](),
			expectedVal: uint32(4294967295),
		},
		{
			name:        "pointer to uint32",
			txt:         "2000",
			typ:         reflect.TypeFor[*uint32](),
			expectedVal: ptr(uint32(2000)),
		},
		{
			name:        "uint64",
			txt:         "9223372036854775808",
			typ:         reflect.TypeFor[uint64](),
			expectedVal: uint64(9223372036854775808),
		},
		{
			name:        "pointer to uint64",
			txt:         "3000",
			typ:         reflect.TypeFor[*uint64](),
			expectedVal: ptr(uint64(3000)),
		},
		{
			name:        "int64",
			txt:         "1234567890",
			typ:         reflect.TypeFor[int64](),
			expectedVal: int64(1234567890),
		},
		{
			name:        "pointer to int64",
			txt:         "-1234567890",
			typ:         reflect.TypeFor[*int64](),
			expectedVal: ptr(int64(-1234567890)),
		},
		{
			name:        "float32",
			txt:         "1.23",
			typ:         reflect.TypeFor[float32](),
			expectedVal: float32(1.23),
		},
		{
			name:        "pointer to float32",
			txt:         "-1.23",
			typ:         reflect.TypeFor[*float32](),
			expectedVal: ptr(float32(-1.23)),
		},
		{
			name:        "float64",
			txt:         "1.23456",
			typ:         reflect.TypeFor[float64](),
			expectedVal: float64(1.23456),
		},
		{
			name:        "pointer to float64",
			txt:         "-1.23456",
			typ:         reflect.TypeFor[*float64](),
			expectedVal: ptr(float64(-1.23456)),
		},
		{
			name:        "complex64",
			txt:         "1.2+3i",
			typ:         reflect.TypeFor[complex64](),
			expectedVal: complex64(1.2 + 3i),
		},
		{
			name:        "pointer to complex64",
			txt:         "1.2+3i",
			typ:         reflect.TypeFor[*complex64](),
			expectedVal: ptr(complex64(1.2 + 3i)),
		},
		{
			name:        "complex128",
			txt:         "1.2+3i",
			typ:         reflect.TypeFor[complex128](),
			expectedVal: complex128(1.2 + 3i),
		},
		{
			name:        "pointer to complex128",
			txt:         "1.2+3i",
			typ:         reflect.TypeFor[*complex128](),
			expectedVal: ptr(complex128(1.2 + 3i)),
		},
		{
			name:        "decimal",
			txt:         "123.45",
			typ:         reflect.TypeFor[decimal.Decimal](),
			expectedVal: decimal.NewFromFloat(123.45),
		},
		{
			name:        "duration",
			txt:         "10s",
			typ:         reflect.TypeFor[time.Duration](),
			expectedVal: 10 * time.Second,
		},
		{
			name:        "time",
			txt:         "2024-01-01T12:00:00Z",
			typ:         reflect.TypeFor[time.Time](),
			expectedVal: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		},
		{
			name:        "struct",
			txt:         `{"Name":"test", "Value":123}`,
			typ:         reflect.TypeFor[obj](),
			expectedVal: obj{Name: "test", Value: 123},
		},
		{
			name:        "struct pointer",
			txt:         `{"Name":"test", "Value":123}`,
			typ:         reflect.TypeFor[*obj](),
			expectedVal: &obj{Name: "test", Value: 123},
		},
		{
			name:        "unmarshaler (enum)",
			txt:         `Test1`,
			typ:         reflect.TypeFor[MyEnumer](),
			expectedVal: Test1,
		},
		{
			name:        "unmarshaler (enum pointer)",
			txt:         `Test2`,
			typ:         reflect.TypeFor[*MyEnumer](),
			expectedVal: Ptr(Test2),
		},
		{
			name:        "slice pointer",
			txt:         `1,2,3`,
			typ:         reflect.TypeFor[*[]int](),
			expectedVal: Ptr([]int{1, 2, 3}),
		},
		{
			name:        "int slice",
			txt:         `1,2,3`,
			typ:         reflect.TypeFor[[]int](),
			expectedVal: []int{1, 2, 3},
		},
		{
			name:        "string pointer slice",
			txt:         `1,2,3`,
			typ:         reflect.TypeFor[[]*string](),
			expectedVal: []*string{Ptr("1"), Ptr("2"), Ptr("3")},
		},
		{
			name:        "decimal array",
			txt:         `1.1,2.2,3.3`,
			typ:         reflect.TypeFor[[3]decimal.Decimal](),
			expectedVal: [3]decimal.Decimal{decimal.NewFromFloat(1.1), decimal.NewFromFloat(2.2), decimal.NewFromFloat(3.3)},
		},
		{
			name:        "empty array",
			txt:         "",
			typ:         reflect.TypeFor[[3]int](),
			expectedVal: [3]int{},
		},
		{
			name:        "empty array pointer",
			txt:         "",
			typ:         reflect.TypeFor[*[3]int](),
			expectedVal: &[3]int{},
		},
		{
			name:        "decimal slice",
			txt:         `1.1,2.2,3.3`,
			typ:         reflect.TypeFor[[]decimal.Decimal](),
			expectedVal: []decimal.Decimal{decimal.NewFromFloat(1.1), decimal.NewFromFloat(2.2), decimal.NewFromFloat(3.3)},
		},
		{
			name:        "empty slice",
			txt:         "",
			typ:         reflect.TypeFor[[]int](),
			expectedVal: []int{},
		},
		{
			name:        "empty pointer to slice",
			txt:         "",
			typ:         reflect.TypeFor[*[]int](),
			expectedVal: &[]int{},
		},
		{
			name:        "slice with empty elements",
			txt:         "a,,b",
			typ:         reflect.TypeFor[[]string](),
			expectedVal: []string{"a", "", "b"},
		},
		{
			name:        "byte slice from hex",
			txt:         "0x68656c6c6f", // "hello"
			typ:         reflect.TypeFor[[]byte](),
			expectedVal: []byte("hello"),
		},
		{
			name:        "pointer to byte slice from hex",
			txt:         "0x776f726c64", // "world"
			typ:         reflect.TypeFor[*[]byte](),
			expectedVal: Ptr([]byte("world")),
		},
		{
			name:        "string map",
			txt:         `a=1,b=2,c=3`,
			typ:         reflect.TypeFor[map[string]int](),
			expectedVal: map[string]int{"a": 1, "b": 2, "c": 3},
		},
		{
			name:        "int map",
			txt:         `100=1,101=2,102=3`,
			typ:         reflect.TypeFor[map[int]int](),
			expectedVal: map[int]int{100: 1, 101: 2, 102: 3},
		},
		{
			name:        "map with pointer values",
			txt:         `a=1,b=2`,
			typ:         reflect.TypeFor[map[string]*int](),
			expectedVal: map[string]*int{"a": Ptr(1), "b": Ptr(2)},
		},
		{
			name:        "map with duration values",
			txt:         `short=5s,long=1m`,
			typ:         reflect.TypeFor[map[string]time.Duration](),
			expectedVal: map[string]time.Duration{"short": 5 * time.Second, "long": 1 * time.Minute},
		},
		{
			name:        "empty map",
			txt:         "",
			typ:         reflect.TypeFor[map[string]int](),
			expectedVal: map[string]int{},
		},
		{
			name:        "empty pointer to map",
			txt:         "",
			typ:         reflect.TypeFor[*map[string]int](),
			expectedVal: &map[string]int{},
		},
		{
			name:        "map with empty value",
			txt:         "a=1,b=",
			typ:         reflect.TypeFor[map[string]string](),
			expectedVal: map[string]string{"a": "1", "b": ""},
		},
		{
			name:        "malformed map",
			txt:         "a,b=5",
			typ:         reflect.TypeFor[map[string]string](),
			expectedVal: map[string]string{"b": "5"},
		},
		{
			name:        "malformed map 2",
			txt:         "5",
			typ:         reflect.TypeFor[map[string]string](),
			expectedVal: map[string]string{},
		},
		{
			name:        "net.IP",
			txt:         "192.168.1.1",
			typ:         reflect.TypeFor[net.IP](),
			expectedVal: net.IP{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xff, 0xc0, 0xa8, 0x1, 0x1},
		},
		{
			name:        "net.HardwareAddr",
			txt:         "AA:BB:CC:DD:EE:FF",
			typ:         reflect.TypeFor[net.HardwareAddr](),
			expectedVal: net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		},
		{
			name:        "net.IPNet",
			txt:         "192.168.1.0/24",
			typ:         reflect.TypeFor[net.IPNet](),
			expectedVal: net.IPNet{IP: net.IP{0xc0, 0xa8, 0x1, 0x0}, Mask: net.IPMask{0xff, 0xff, 0xff, 0x0}},
		},
		{
			name:        "struct slice",
			txt:         "{\"name\":\"Gemini\",\"age\":1},{\"name\":\"Code Assist\",\"age\":2}",
			typ:         reflect.TypeFor[[]jsonStruct](),
			expectedVal: []jsonStruct{{Name: "Gemini", Age: 1}, {Name: "Code Assist", Age: 2}},
		},
		{
			name:        "struct array",
			txt:         "{\"name\":\"Gemini\",\"age\":1},{\"name\":\"Code Assist\",\"age\":2}",
			typ:         reflect.TypeFor[[2]jsonStruct](),
			expectedVal: [2]jsonStruct{{Name: "Gemini", Age: 1}, {Name: "Code Assist", Age: 2}},
		},
		{
			name:        "struct map",
			txt:         "a={\"name\":\"Gemini\",\"age\":1},b={\"name\":\"Code Assist\",\"age\":2}",
			typ:         reflect.TypeFor[map[string]jsonStruct](),
			expectedVal: map[string]jsonStruct{"a": {Name: "Gemini", Age: 1}, "b": {Name: "Code Assist", Age: 2}},
		},
		{
			name:        "hex slice",
			txt:         "0x010203",
			typ:         reflect.TypeFor[[]byte](),
			expectedVal: []byte{1, 2, 3},
		},
		{
			name:        "empty hex slice",
			txt:         "",
			typ:         reflect.TypeFor[[]byte](),
			expectedVal: []byte{},
		},
		{
			name:        "hex slice pointer",
			txt:         "0x010203",
			typ:         reflect.TypeFor[*[]byte](),
			expectedVal: &[]byte{1, 2, 3},
		},
		{
			name:        "hex array",
			txt:         "0x010203",
			typ:         reflect.TypeFor[[3]byte](),
			expectedVal: [3]byte{1, 2, 3},
		},
		{
			name:        "empty hex array",
			txt:         "",
			typ:         reflect.TypeFor[[3]byte](),
			expectedVal: [3]byte{0, 0, 0},
		},
		{
			name:        "partial hex array",
			txt:         "0x0102",
			typ:         reflect.TypeFor[[3]byte](),
			expectedVal: [3]byte{1, 2, 0},
		},
		{
			name:        "hex array pointer",
			txt:         "0x010203",
			typ:         reflect.TypeFor[*[3]byte](),
			expectedVal: &[3]byte{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := lookup.Parse(tt.txt, tt.typ)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedVal, result)
		})
	}
}

func TestParse_Errors(t *testing.T) {
	t.Parallel()

	type obj struct {
		Name  string
		Value float64
	}

	tests := []struct {
		name        string
		txt         string
		typ         reflect.Type
		expectedVal any
		expectErr   string
	}{
		// Error cases
		{
			name:      "unsupported type",
			txt:       "some value",
			typ:       reflect.TypeFor[uintptr](),
			expectErr: "unsupported data type",
		},
		{
			name:      "int8 overflow",
			txt:       "128",
			typ:       reflect.TypeFor[int8](),
			expectErr: "value out of range",
		},
		{
			name:      "uint8 overflow",
			txt:       "256",
			typ:       reflect.TypeFor[uint8](),
			expectErr: "value out of range",
		},
		{
			name:      "invalid duration",
			txt:       "10xyz",
			typ:       reflect.TypeFor[time.Duration](),
			expectErr: "unknown unit",
		},
		{
			name:      "invalid time",
			txt:       "not-a-time",
			typ:       reflect.TypeFor[time.Time](),
			expectErr: "cannot parse",
		},
		{
			name:      "unmarshaler error",
			txt:       "any",
			typ:       reflect.TypeFor[errorUnmarshaler](),
			expectErr: "assert.AnError",
		},
		{
			name:      "slice with invalid element",
			txt:       "1,two,3",
			typ:       reflect.TypeFor[[]int](),
			expectErr: "invalid syntax",
		},
		{
			name:      "map with invalid key",
			txt:       "a=1,two=2",
			typ:       reflect.TypeFor[map[int]int](),
			expectErr: "invalid syntax",
		},
		{
			name:      "map with invalid value",
			txt:       "1=a,2=b",
			typ:       reflect.TypeFor[map[int]int](),
			expectErr: "invalid syntax",
		},
		{
			name:      "malformed map",
			txt:       "1=a,2",
			typ:       reflect.TypeFor[map[int]int](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid json for struct",
			txt:       `{"Name":"test", "Value":}`,
			typ:       reflect.TypeFor[obj](),
			expectErr: "invalid character",
		},
		{
			name:      "invalid bool",
			txt:       "not-a-bool",
			typ:       reflect.TypeFor[bool](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid int",
			txt:       "abc",
			typ:       reflect.TypeFor[int](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid int8",
			txt:       "abc",
			typ:       reflect.TypeFor[int8](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid int16",
			txt:       "abc",
			typ:       reflect.TypeFor[int16](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid int32",
			txt:       "abc",
			typ:       reflect.TypeFor[int32](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid int64",
			txt:       "abc",
			typ:       reflect.TypeFor[int64](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid uint",
			txt:       "-1",
			typ:       reflect.TypeFor[uint](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid uint8",
			txt:       "abc",
			typ:       reflect.TypeFor[uint8](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid uint16",
			txt:       "abc",
			typ:       reflect.TypeFor[uint16](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid uint32",
			txt:       "abc",
			typ:       reflect.TypeFor[uint32](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid uint64",
			txt:       "abc",
			typ:       reflect.TypeFor[uint64](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid float32",
			txt:       "abc",
			typ:       reflect.TypeFor[float32](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid float64",
			txt:       "abc",
			typ:       reflect.TypeFor[float64](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid complex64",
			txt:       "abc",
			typ:       reflect.TypeFor[complex64](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid complex128",
			txt:       "abc",
			typ:       reflect.TypeFor[complex128](),
			expectErr: "invalid syntax",
		},
		{
			name:      "invalid hex slice",
			txt:       "0xzz",
			typ:       reflect.TypeFor[[]byte](),
			expectErr: "invalid byte",
		},
		{
			name:      "invalid hex array",
			txt:       "0xzz",
			typ:       reflect.TypeFor[[3]byte](),
			expectErr: "invalid byte",
		},
		{
			name:      "net.HardwareAddr error",
			txt:       "invalid-mac",
			typ:       reflect.TypeFor[net.HardwareAddr](),
			expectErr: "invalid MAC address",
		},
		{
			name:      "net.IPNet error",
			txt:       "invalid-cidr",
			typ:       reflect.TypeFor[net.IPNet](),
			expectErr: "invalid CIDR address",
		},
		{
			name:      "array length mismatch",
			txt:       "1,2,3",
			typ:       reflect.TypeFor[[2]int](),
			expectErr: "expected 2 elements, got 3",
		},
		{
			name:      "TextUnmarshaler error",
			txt:       "invalid",
			typ:       reflect.TypeFor[unmarshaler](),
			expectErr: "invalid syntax",
		},
		{
			name:      "TextUnmarshaler pointer error",
			txt:       "invalid",
			typ:       reflect.TypeFor[*unmarshaler](),
			expectErr: "invalid syntax",
		},
		{
			name:      "YAMLUnmarshaler error",
			txt:       "invalid",
			typ:       reflect.TypeFor[yamlUnmarshaler](),
			expectErr: "invalid syntax",
		},
		{
			name:      "YAMLUnmarshaler pointer error",
			txt:       "invalid",
			typ:       reflect.TypeFor[*yamlUnmarshaler](),
			expectErr: "invalid syntax",
		},
		{
			name:      "array element error",
			txt:       "1,invalid",
			typ:       reflect.TypeFor[[2]int](),
			expectErr: "invalid syntax",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// All these tests expect an error
			result, err := lookup.Parse(tt.txt, tt.typ)
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tt.expectErr)
			}
			assert.Nil(t, result)
		})
	}
}

type customParserHookType struct {
	Data string
}

func TestParse_WithHook(t *testing.T) {
	f := func(txt string) (any, error) {
		if !strings.HasPrefix(txt, "custom:") {
			return nil, errors.New("invalid format for custom parser")
		}
		return customParserHookType{Data: strings.TrimPrefix(txt, "custom:")}, nil
	}

	hook := lookup.NewParserHook(reflect.TypeOf(customParserHookType{}), f)

	t.Run("successful parse with hook", func(t *testing.T) {
		result, err := lookup.Parse("custom:my-data", reflect.TypeOf(customParserHookType{}), hook)
		require.NoError(t, err)
		assert.Equal(t, customParserHookType{Data: "my-data"}, result)
	})

	t.Run("error on parse with hook", func(t *testing.T) {
		_, err := lookup.Parse("invalid-data", reflect.TypeOf(customParserHookType{}), hook)
		require.Error(t, err)
		assert.EqualError(t, err, "invalid format for custom parser")
	})

	t.Run("no hook for type falls through", func(t *testing.T) {
		// The hook is for customParserHookType, not string.
		// This should fall through to the standard string parsing.
		result, err := lookup.Parse("just a string", reflect.TypeOf(""), hook)
		require.NoError(t, err)
		assert.Equal(t, "just a string", result)
	})
}
