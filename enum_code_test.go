// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package lookup_test

import (
	"encoding/json"
	"fmt"
	"strings"
)

const _MyEnumerName = "Test1Test2Test3"

var _MyEnumerIndex = [...]uint8{0, 5, 10, 15}

const _MyEnumerLowerName = "test1test2test3"

func (i MyEnumer) String() string {
	if i < 0 || i >= MyEnumer(len(_MyEnumerIndex)-1) {
		return fmt.Sprintf("MyEnumer(%d)", i)
	}
	return _MyEnumerName[_MyEnumerIndex[i]:_MyEnumerIndex[i+1]]
}

// An "invalid array index" compiler error signifies that the constant values have changed.
// Re-run the stringer command to generate them again.
func _MyEnumerNoOp() {
	var x [1]struct{}
	_ = x[Test1-(0)]
	_ = x[Test2-(1)]
	_ = x[Test3-(2)]
}

var _MyEnumerValues = []MyEnumer{Test1, Test2, Test3}

var _MyEnumerNameToValueMap = map[string]MyEnumer{
	_MyEnumerName[0:5]:        Test1,
	_MyEnumerLowerName[0:5]:   Test1,
	_MyEnumerName[5:10]:       Test2,
	_MyEnumerLowerName[5:10]:  Test2,
	_MyEnumerName[10:15]:      Test3,
	_MyEnumerLowerName[10:15]: Test3,
}

var _MyEnumerNames = []string{
	_MyEnumerName[0:5],
	_MyEnumerName[5:10],
	_MyEnumerName[10:15],
}

// MyEnumerString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func MyEnumerString(s string) (MyEnumer, error) {
	if val, ok := _MyEnumerNameToValueMap[s]; ok {
		return val, nil
	}

	if val, ok := _MyEnumerNameToValueMap[strings.ToLower(s)]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to MyEnumer values", s)
}

// MyEnumerValues returns all values of the enum
func MyEnumerValues() []MyEnumer {
	return _MyEnumerValues
}

// MyEnumerStrings returns a slice of all String values of the enum
func MyEnumerStrings() []string {
	strs := make([]string, len(_MyEnumerNames))
	copy(strs, _MyEnumerNames)
	return strs
}

// IsAMyEnumer returns "true" if the value is listed in the enum definition. "false" otherwise
func (i MyEnumer) IsAMyEnumer() bool {
	for _, v := range _MyEnumerValues {
		if i == v {
			return true
		}
	}
	return false
}

// MarshalJSON implements the json.Marshaler interface for MyEnumer
func (i MyEnumer) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface for MyEnumer
func (i *MyEnumer) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("MyEnumer should be a string, got %s", data)
	}

	var err error
	*i, err = MyEnumerString(s)
	return err
}

// MarshalText implements the encoding.TextMarshaler interface for MyEnumer
func (i MyEnumer) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for MyEnumer
func (i *MyEnumer) UnmarshalText(text []byte) error {
	var err error
	*i, err = MyEnumerString(string(text))
	return err
}

// MarshalYAML implements a YAML Marshaler for MyEnumer
func (i MyEnumer) MarshalYAML() (interface{}, error) {
	return i.String(), nil
}

// UnmarshalYAML implements a YAML Unmarshaler for MyEnumer
func (i *MyEnumer) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	var err error
	*i, err = MyEnumerString(s)
	return err
}
