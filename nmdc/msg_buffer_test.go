package nmdc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuffer(t *testing.T) {
	var b Buffer
	err := b.WriteMessage(
		myInfo("user"),
		&ChatMessage{Name: "user", Text: "hi"},
	)
	require.NoError(t, err)

	data, err := b.BytesFor(DefaultEncoding())
	require.NoError(t, err)
	require.Equal(t,
		`$MyINFO $ALL user RU<ApexDC++ V:0.4.0,M:P,H:27/1/3,S:92,L:512>$ $LAN(T3)K$example@example.com$1234$|<user> hi|`,
		string(data),
	)
}
