package vhost

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultNotFoundPageDoesNotIdentifySoftware(t *testing.T) {
	body, err := io.ReadAll(NotFoundResponse().Body)
	require.NoError(t, err)
	text := strings.ToLower(string(body))
	require.NotContains(t, text, "frp")
	require.NotContains(t, text, "github.com")
	require.Empty(t, NotFoundResponse().Header.Get("Server"))
}
