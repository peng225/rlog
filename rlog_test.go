package rlog

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getRightN(s string, n int) string {

	return s[max(0, len(s)-n):]
}

func TestRlog(t *testing.T) {
	b := new(bytes.Buffer)
	logger := slog.New(NewRawTextHandler(b, nil))
	require.NotNil(t, logger)

	logger.Info("test")
	expected := "test\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	b.Reset()
	logger.Info("test\nnewline")
	expected = "test\nnewline\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	b.Reset()
	logger.Info("test", "key", "value")
	expected = "test (key=value)\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	b.Reset()
	logger.Info("test", "key1", "value1", slog.String("key2", "value2"))
	expected = "test (key1=value1, key2=value2)\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	b.Reset()
	logger.Info("test", slog.Group("outerKey", "innerKey", "value"))
	expected = "test (outerKey=(innerKey=value))\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	b.Reset()
	logger.Info("test", slog.Group("outerKey", "innerKey1", "value1", "innerKey2", 2))
	expected = "test (outerKey=(innerKey1=value1, innerKey2=2))\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	b.Reset()
	logger.Info("test", slog.Group("outerKey",
		slog.Group("middleKey", "innerKey1", 1, "innerKey2", 2)))
	expected = "test (outerKey=(middleKey=(innerKey1=1, innerKey2=2)))\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))
}

func TestRlogLevel(t *testing.T) {
	b := new(bytes.Buffer)
	logger := slog.New(NewRawTextHandler(b, &slog.HandlerOptions{
		Level: slog.LevelWarn,
	}))
	require.NotNil(t, logger)

	cases := []struct {
		name        string
		logFunc     func(msg string, args ...any)
		expectedLog string
	}{
		{
			name:        "Debug not printed",
			logFunc:     logger.Debug,
			expectedLog: "",
		},
		{
			name:        "Info not printed",
			logFunc:     logger.Info,
			expectedLog: "",
		},
		{
			name:        "Warn printed",
			logFunc:     logger.Warn,
			expectedLog: "test\n",
		},
		{
			name:        "Error printed",
			logFunc:     logger.Error,
			expectedLog: "test\n",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			tt.logFunc("test")
			assert.Equal(t, tt.expectedLog, getRightN(b.String(), len(tt.expectedLog)))
		})
	}
}
