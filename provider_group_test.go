// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderGroup(t *testing.T) {
	t.Parallel()
	pg := NewProviderGroup("test-group", NewYAMLProviderFromBytes([]byte(`id: test`)))
	assert.Equal(t, "test-group", pg.Name())
	assert.Equal(t, "test", pg.Get("id").AsString())
	// TODO this should not require a cast GFM-74
	assert.Empty(t, pg.(providerGroup).RegisterChangeCallback(Root, nil))
	assert.Nil(t, pg.(providerGroup).UnregisterChangeCallback(Root))
}

func TestProviderGroupScope(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{"hello": map[string]int{"world": 42}}
	pg := NewProviderGroup("test-group", NewStaticProvider(data))
	assert.Equal(t, 42, pg.Get("hello").Get("world").AsInt())
}

func TestCallbacks_WithDynamicProvider(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{"hello.world": 42}
	mock := NewProviderGroup("with-dynamic", NewStaticProvider(data), NewMockDynamicProvider(data))
	assert.Equal(t, "with-dynamic", mock.Name())

	require.NoError(t, mock.RegisterChangeCallback("mockcall", nil))
	assert.EqualError(t,
		mock.RegisterChangeCallback("mockcall", nil),
		"callback already registered for the key: mockcall")

	assert.EqualError(t,
		mock.UnregisterChangeCallback("mock"),
		"there is no registered callback for token: mock")
}

func TestCallbacks_WithoutDynamicProvider(t *testing.T) {
	t.Parallel()
	data := map[string]interface{}{"hello.world": 42}
	mock := NewProviderGroup("with-dynamic", NewStaticProvider(data))
	assert.Equal(t, "with-dynamic", mock.Name())
	assert.NoError(t, mock.RegisterChangeCallback("mockcall", nil))
	assert.NoError(t, mock.UnregisterChangeCallback("mock"))
}

func TestCallbacks_WithScopedProvider(t *testing.T) {
	t.Parallel()
	mock := &MockDynamicProvider{}
	mock.Set("uber.fx", "go-lang")
	scope := NewScopedProvider("uber", mock)

	callCount := 0
	cb := func(key string, provider string, configdata interface{}) {
		callCount++
	}

	require.NoError(t, scope.RegisterChangeCallback("fx", cb))
	mock.Set("uber.fx", "register works!")

	val := scope.Get("fx").AsString()
	require.Equal(t, "register works!", val)
	assert.Equal(t, 1, callCount)

	require.NoError(t, scope.UnregisterChangeCallback("fx"))
	mock.Set("uber.fx", "unregister works too!")

	val = scope.Get("fx").AsString()
	require.Equal(t, "unregister works too!", val)
	assert.Equal(t, 1, callCount)
}

func TestScope_WithGetFromValue(t *testing.T) {
	t.Parallel()
	mock := &MockDynamicProvider{}
	mock.Set("uber.fx", "go-lang")
	scope := NewScopedProvider("", mock)
	require.Equal(t, "go-lang", scope.Get("uber.fx").AsString())
	require.False(t, scope.Get("uber").HasValue())

	base := scope.Get("uber")
	require.Equal(t, "go-lang", base.Get("fx").AsString())
	require.False(t, base.Get("").HasValue())

	uber := base.Get(Root)
	require.Equal(t, "go-lang", uber.Get("fx").AsString())
	require.False(t, uber.Get("").HasValue())

	fx := uber.Get("fx")
	require.Equal(t, "go-lang", fx.Get("").AsString())
	require.False(t, fx.Get("fx").HasValue())
}

func TestProviderGroupScopingValue(t *testing.T) {
	t.Parallel()
	fst := []byte(`
logging:`)

	snd := []byte(`
logging:
  enabled: true
`)
	pg := NewProviderGroup("group", NewYAMLProviderFromBytes(snd), NewYAMLProviderFromBytes(fst))
	assert.True(t, pg.Get("logging").Get("enabled").AsBool())
}

func TestProviderGroup_GetChecksAllProviders(t *testing.T) {
	t.Parallel()

	pg := NewProviderGroup("test-group",
		NewStaticProvider(map[string]string{"name": "test", "desc": "test"}),
		NewStaticProvider(map[string]string{"owner": "tst@example.com", "name": "fx"}))

	require.NotNil(t, pg)

	var svc map[string]string
	require.NoError(t, pg.Get(Root).Populate(&svc))
	assert.Equal(t, map[string]string{"name": "fx", "owner": "tst@example.com", "desc": "test"}, svc)
}
