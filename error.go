// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package lookup

import "errors"

var (
	ErrUnsupportedMap = errors.New("only maps with string keys are supported")
	ErrNotMap         = errors.New("field is not a map")
	ErrNotSlice       = errors.New("field is not an array or slice")
)

type NotFoundError struct {
	Name string
}

func (e *NotFoundError) Error() string {
	return "field not found: " + e.Name
}
