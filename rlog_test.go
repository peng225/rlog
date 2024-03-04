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
	logger := slog.New(NewRawTextHandler(b, &HandlerOptions{
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

func TestRlogWithAttrsAndGroups(t *testing.T) {
	b := new(bytes.Buffer)
	logger := slog.New(NewRawTextHandler(b, nil))
	require.NotNil(t, logger)

	// Empty group should be ignored.
	loggerWithGroup := logger.WithGroup("group1")
	loggerWithGroup.Info("test")
	expected := "test\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	// One group.
	b.Reset()
	loggerWithGroup.Info("test", "key", "value")
	expected = "test (group1=(key=value))\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	// The original logger should not be modified.
	b.Reset()
	logger.Info("test", "key", "value")
	expected = "test (key=value)\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	// Two groups.
	b.Reset()
	loggerWithTwoGroups := loggerWithGroup.WithGroup("group2")
	loggerWithTwoGroups.Info("test", "key", "value")
	expected = "test (group1=(group2=(key=value)))\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	// One attr.
	b.Reset()
	loggerWithAttr := logger.With("key1", "value1")
	loggerWithAttr.Info("test")
	expected = "test (key1=value1)\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	// One attr with additional attr.
	b.Reset()
	loggerWithAttr.Info("test", "key2", "value2")
	expected = "test (key1=value1, key2=value2)\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	// Group -> Attr -> Group -> Attr
	b.Reset()
	loggerGAGA := logger.WithGroup("g1").With("k1", "v1").WithGroup("g2").With("k2", "v2")
	loggerGAGA.Info("test")
	expected = "test (g1=(k1=v1, g2=(k2=v2)))\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	// Group -> Attr -> Group -> Attr -> Attr(additional)
	b.Reset()
	loggerGAGA.Info("test", "k3", "v3")
	expected = "test (g1=(k1=v1, g2=(k2=v2, k3=v3)))\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	// Group -> Attr -> Attr -> Group -> Group -> Attr
	b.Reset()
	loggerGAAGGA := logger.WithGroup("g1").
		With("k1", "v1").With("k2", "v2").
		WithGroup("g2").WithGroup("g3").With("k3", "v3")
	loggerGAAGGA.Info("test")
	expected = "test (g1=(k1=v1, k2=v2, g2=(g3=(k3=v3))))\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))

	// Attr -> Group -> Attr -> Group(additional) -> Attr(additional) -> Attr(additional)
	b.Reset()
	loggerAGA := logger.With("k1", "v1").WithGroup("g1").With("k2", "v2")
	loggerAGA.Info("test", slog.Group("g2", "k3", "v3", "k4", "v4"))
	expected = "test (k1=v1, g1=(k2=v2, g2=(k3=v3, k4=v4)))\n"
	assert.Equal(t, expected, getRightN(b.String(), len(expected)))
}
