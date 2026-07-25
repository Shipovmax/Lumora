package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenizeLowercasesAndFiltersShortWords(t *testing.T) {
	tokens := tokenize("Hello, World! A cat sat on a mat.")
	require.Contains(t, tokens, "hello")
	require.Contains(t, tokens, "world")
	require.Contains(t, tokens, "cat")
	require.Contains(t, tokens, "sat")
	require.Contains(t, tokens, "mat")
	require.NotContains(t, tokens, "a")
	require.NotContains(t, tokens, "on")
}

func TestJaccardSimilarity(t *testing.T) {
	a := tokenize("bitcoin price surges today")
	b := tokenize("bitcoin price rallies today")

	similarity := jaccardSimilarity(a, b)
	require.Greater(t, similarity, 0.0)
	require.Less(t, similarity, 1.0)

	require.Equal(t, 0.0, jaccardSimilarity(map[string]struct{}{}, b))
	require.Equal(t, 1.0, jaccardSimilarity(a, a))
}
