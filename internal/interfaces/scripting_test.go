package interfaces

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariableScopeString(t *testing.T) {
	tests := []struct {
		scope VariableScope
		want  string
	}{
		{VariableScopeLocal, "local"},
		{VariableScopeRequest, "request"},
		{VariableScopeCollection, "collection"},
		{VariableScopeEnvironment, "environment"},
		{VariableScopeGlobal, "global"},
		{VariableScopeBuiltin, "builtin"},
		{VariableScope(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.scope.String())
		})
	}
}
