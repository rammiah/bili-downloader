package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJson(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		v := struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}{
			ID:   100,
			Name: "windows",
		}
		res := Json(v)
		require.Equal(t, `{"id":100,"name":"windows"}`, res)

	})

	t.Run("error", func(t *testing.T) {
		v := make(chan int)
		res := Json(v)
		require.Equal(t, "<error>", res)
	})
}
