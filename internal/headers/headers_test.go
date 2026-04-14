package headers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadersParse(t *testing.T) {
	t.Run("Valid single header", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host: localhost:42069\r\n\r\n")

		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		require.NotNil(t, headers)
		assert.Equal(t, "localhost:42069", headers["host"])
		assert.Equal(t, 23, n)
		assert.False(t, done)
	})

	t.Run("Valid single header with extra whitespace", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("HoSt:           localhost:42069    \r\n\r\n")

		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, "localhost:42069", headers["host"])
		assert.Equal(t, 37, n)
		assert.False(t, done)
	})

	t.Run("Valid 2 headers with existing headers", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host: localhost:42069\r\nUser-Agent: curl/7.81.0\r\n\r\n")

		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, "localhost:42069", headers["host"])
		assert.Equal(t, 23, n)
		assert.False(t, done)

		n2, done2, err2 := headers.Parse(data[n:])
		require.NoError(t, err2)
		assert.Equal(t, "curl/7.81.0", headers["user-agent"])
		assert.Equal(t, 25, n2)
		assert.False(t, done2)
	})

	t.Run("Valid done", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("\r\n")

		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, 2, n)
		assert.True(t, done)
	})

	t.Run("Valid duplicate header appends value", func(t *testing.T) {
		headers := NewHeaders()
		headers["set-person"] = "lane-loves-go"
		data := []byte("Set-Person: prime-loves-zig\r\n\r\n")

		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, 29, n)
		assert.False(t, done)
		assert.Equal(t, "lane-loves-go, prime-loves-zig", headers["set-person"])
	})

	t.Run("Invalid spacing header", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("       Host : localhost:42069       \r\n\r\n")

		n, done, err := headers.Parse(data)
		require.Error(t, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	t.Run("Invalid character in header key", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("H©st: localhost:42069\r\n\r\n")

		n, done, err := headers.Parse(data)
		require.Error(t, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})

	t.Run("Incomplete header line needs more data", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host: localhost:42069")

		n, done, err := headers.Parse(data)
		require.NoError(t, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
		assert.Empty(t, headers)
	})

	t.Run("Invalid header missing colon", func(t *testing.T) {
		headers := NewHeaders()
		data := []byte("Host localhost:42069\r\n")

		n, done, err := headers.Parse(data)
		require.Error(t, err)
		assert.Equal(t, 0, n)
		assert.False(t, done)
	})
}
