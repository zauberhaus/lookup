// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package lookup

import "reflect"

type ParserHook struct {
	To    reflect.Type
	Parse func(txt string) (any, error)
}

func NewParserHookFor[T any](f func(txt string) (any, error)) ParserHook {
	t := reflect.TypeFor[T]()
	return NewParserHook(t, f)
}

func NewParserHook(to reflect.Type, f func(txt string) (any, error)) ParserHook {
	return ParserHook{
		To:    to,
		Parse: f,
	}
}
