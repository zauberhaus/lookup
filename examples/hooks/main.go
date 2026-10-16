// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/zauberhaus/lookup"
)

// Secret is an interface, so Set can't parse a string into it on its own.
type Secret interface {
	Reveal() string
}

type PlainSecret struct {
	Value string
}

func (s *PlainSecret) Reveal() string {
	return s.Value
}

type EnvSecret struct {
	Name string
}

func (s *EnvSecret) Reveal() string {
	return "${" + s.Name + "}"
}

type Config struct {
	Password Secret
	Tokens   []Secret
	Keys     map[string]Secret
}

func main() {
	// A hook parses a string into the Secret interface.
	// "env:NAME" refers to an environment variable, anything else is the secret itself.
	hook := lookup.NewParserHookFor[Secret](func(txt string) (any, error) {
		if name, ok := strings.CutPrefix(txt, "env:"); ok {
			return &EnvSecret{Name: name}, nil
		}

		return &PlainSecret{Value: txt}, nil
	})

	var cfg Config

	// 1. Without the hook, a string can't be set to an interface field
	_, err := lookup.Set(&cfg, "password", "s3cr3t")
	fmt.Printf("Without hook: %v\n", err)

	// 2. With the hook, the string is parsed into a Secret
	_, err = lookup.Set(&cfg, "password", "s3cr3t", hook)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Password: %T %v\n", cfg.Password, cfg.Password.Reveal())

	// 3. Hooks also work for slice and map elements
	_, err = lookup.Set(&cfg, "tokens[0]", "env:API_TOKEN", hook)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Token: %T %v\n", cfg.Tokens[0], cfg.Tokens[0].Reveal())

	_, err = lookup.Set(&cfg, "keys[db]", "env:DB_KEY", hook)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Key: %T %v\n", cfg.Keys["db"], cfg.Keys["db"].Reveal())

	// 4. A value that already implements the interface needs no hook
	_, err = lookup.Set(&cfg, "password", &PlainSecret{Value: "direct"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Direct: %T %v\n", cfg.Password, cfg.Password.Reveal())
}
