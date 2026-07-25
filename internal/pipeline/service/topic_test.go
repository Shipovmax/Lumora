package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/pipeline/domain"
)

func TestClassifyTopic(t *testing.T) {
	require.Equal(t, domain.TopicAI, classifyTopic("OpenAI releases GPT-5", "Anthropic's Claude also updated"))
	require.Equal(t, domain.TopicCrypto, classifyTopic("Bitcoin surges", "ETH follows"))
	require.Equal(t, domain.TopicEconomy, classifyTopic("Inflation rises", "Federal Reserve responds"))
	require.Equal(t, domain.TopicWorld, classifyTopic("President meets government officials", "Sanctions discussed"))
	require.Equal(t, domain.TopicOther, classifyTopic("Local bakery wins award", "Best bread in town"))
}
