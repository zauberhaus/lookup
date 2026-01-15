// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package main

import (
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/zauberhaus/lookup"
)

type CustomType struct {
	Data string
}

func main() {
	// 1. Parse a duration string into time.Duration
	val, err := lookup.Parse("10s", reflect.TypeFor[time.Duration]())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Duration: %v\n", val)

	// 2. Parse a comma-separated string into a slice of ints
	val, err = lookup.Parse("1,2,3,4", reflect.TypeFor[[]int]())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Slice: %v\n", val)

	// 3. Use a custom parser hook for a specific type
	hook := lookup.NewParserHook(reflect.TypeFor[CustomType](), func(input string) (any, error) {
		return CustomType{Data: "parsed: " + input}, nil
	})

	val, err = lookup.Parse("some input", reflect.TypeFor[CustomType](), hook)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Custom: %+v\n", val)
}
