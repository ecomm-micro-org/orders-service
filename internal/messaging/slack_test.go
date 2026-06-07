package messaging

import (
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlackMessengerRejectsNilClient(t *testing.T) {
	_, err := NewSlackMessenger(nil, "channel")
	require.Error(t, err)
}

func TestNewSlackMessengerStoresDependencies(t *testing.T) {
	client := slack.New("token")
	messenger, err := NewSlackMessenger(client, "channel-id")
	require.NoError(t, err)
	require.NotNil(t, messenger)
	assert.Same(t, client, messenger.slackClient)
	assert.Equal(t, "channel-id", messenger.slackChannel)
}
