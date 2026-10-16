// Copyright 2026 Zauberhaus
// Licensed to Zauberhaus under one or more agreements.
// Zauberhaus licenses this file to you under the Apache 2.0 License.
// See the LICENSE file in the project root for more information.

package lookup_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zauberhaus/lookup"
)

type hookSecret interface {
	Reveal() string
}

type textSecret struct {
	text string
}

func (s *textSecret) Reveal() string {
	return s.text
}

type hookConfig struct {
	Pin     hookSecret
	Any     any
	Pins    []hookSecret
	Secrets map[string]hookSecret
	Nested  hookNested
}

type hookNested struct {
	Pin hookSecret
}

var secretHook = lookup.NewParserHookFor[hookSecret](func(txt string) (any, error) {
	return &textSecret{text: txt}, nil
})

func TestSet_HookSetsAnInterfaceField(t *testing.T) {
	var cfg hookConfig

	_, err := lookup.Set(&cfg, "pin", "1234", secretHook)
	require.NoError(t, err)
	require.NotNil(t, cfg.Pin)
	assert.Equal(t, "1234", cfg.Pin.Reveal())

	_, err = lookup.Set(&cfg, "nested.pin", "5678", secretHook)
	require.NoError(t, err)
	require.NotNil(t, cfg.Nested.Pin)
	assert.Equal(t, "5678", cfg.Nested.Pin.Reveal())
}

func TestSet_HookTakesAStringPointer(t *testing.T) {
	var cfg hookConfig

	txt := "1234"

	_, err := lookup.Set(&cfg, "pin", &txt, secretHook)
	require.NoError(t, err)
	assert.Equal(t, "1234", cfg.Pin.Reveal())
}

func TestSet_HookSetsSliceAndMapElements(t *testing.T) {
	var cfg hookConfig

	_, err := lookup.Set(&cfg, "pins[0]", "first", secretHook)
	require.NoError(t, err)
	require.Len(t, cfg.Pins, 1)
	assert.Equal(t, "first", cfg.Pins[0].Reveal())

	_, err = lookup.Set(&cfg, "secrets[db]", "second", secretHook)
	require.NoError(t, err)
	require.Contains(t, cfg.Secrets, "db")
	assert.Equal(t, "second", cfg.Secrets["db"].Reveal())
}

func TestSet_ValueImplementingTheInterfaceNeedsNoHook(t *testing.T) {
	var cfg hookConfig

	_, err := lookup.Set(&cfg, "pin", &textSecret{text: "direct"})
	require.NoError(t, err)
	assert.Equal(t, "direct", cfg.Pin.Reveal())

	_, err = lookup.Set(&cfg, "any", "text", secretHook)
	require.NoError(t, err)
	assert.Equal(t, "text", cfg.Any, "a string is an any, the hook isn't for it")
}

func TestSet_WithoutAHookTheErrorDoesNotShowTheValue(t *testing.T) {
	var cfg hookConfig

	_, err := lookup.Set(&cfg, "pin", "s3cr3t")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "s3cr3t")
	assert.Contains(t, err.Error(), "doesn't implement")
}

func TestSet_HookResultMustImplementTheInterface(t *testing.T) {
	var cfg hookConfig

	wrong := lookup.NewParserHookFor[hookSecret](func(txt string) (any, error) {
		return txt, nil
	})

	_, err := lookup.Set(&cfg, "pin", "1234", wrong)
	assert.Error(t, err)
	assert.Nil(t, cfg.Pin)
}
