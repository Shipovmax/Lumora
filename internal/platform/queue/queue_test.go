package queue_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Shipovmax/Lumora/internal/platform/queue"
)

func TestNewPipelineProcessTask(t *testing.T) {
	task, err := queue.NewPipelineProcessTask([]string{"post-1", "post-2"})
	require.NoError(t, err)
	require.Equal(t, queue.TypePipelineProcess, task.Type())

	var payload queue.PipelineProcessPayload
	require.NoError(t, json.Unmarshal(task.Payload(), &payload))
	require.Equal(t, []string{"post-1", "post-2"}, payload.PostIDs)
}

func TestNewBriefingBuildTask(t *testing.T) {
	task, err := queue.NewBriefingBuildTask("user-1", "morning")
	require.NoError(t, err)
	require.Equal(t, queue.TypeBriefingBuild, task.Type())

	var payload queue.BriefingBuildPayload
	require.NoError(t, json.Unmarshal(task.Payload(), &payload))
	require.Equal(t, "user-1", payload.UserID)
	require.Equal(t, "morning", payload.Type)
}

func TestNewBriefingDispatchTask(t *testing.T) {
	task, err := queue.NewBriefingDispatchTask("evening")
	require.NoError(t, err)
	require.Equal(t, queue.TypeBriefingDispatch, task.Type())

	var payload queue.BriefingDispatchPayload
	require.NoError(t, json.Unmarshal(task.Payload(), &payload))
	require.Equal(t, "evening", payload.Type)
}

func TestNewNotificationPushTask(t *testing.T) {
	task, err := queue.NewNotificationPushTask("user-1", "Title", "Body")
	require.NoError(t, err)
	require.Equal(t, queue.TypeNotificationPush, task.Type())

	var payload queue.NotificationPushPayload
	require.NoError(t, json.Unmarshal(task.Payload(), &payload))
	require.Equal(t, "user-1", payload.UserID)
	require.Equal(t, "Title", payload.Title)
	require.Equal(t, "Body", payload.Body)
}
