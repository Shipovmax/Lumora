package provider

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/ai/domain"
)

func TestBuildPromptIncludesEventAndUserContext(t *testing.T) {
	prompt := buildPrompt(domain.EventInput{
		Title:   "Something happened",
		Topic:   "ai",
		Content: "Full event content",
	}, "Interested in deep tech")

	require.True(t, strings.Contains(prompt, "Something happened"))
	require.True(t, strings.Contains(prompt, "Full event content"))
	require.True(t, strings.Contains(prompt, "Interested in deep tech"))
}

func TestBuildPromptHandlesEmptyUserContext(t *testing.T) {
	prompt := buildPrompt(domain.EventInput{Title: "T", Topic: "world", Content: "C"}, "")
	require.True(t, strings.Contains(prompt, "не указан"))
}
