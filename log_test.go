package docradle

import (
	"bytes"
	"context"
	"github.com/rs/zerolog"
	"gocloud.dev/pubsub"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	zerolog.TimestampFunc = func() time.Time {
		return time.Date(2020, time.January, 25, 10, 0, 0, 0, time.UTC)
	}
}

func TestLog_NoStructuredConsole(t *testing.T) {
	var buffer bytes.Buffer
	logger, err := NewLogger(context.Background(), StdOut, &buffer, "info", LogConfig{
		Structured:   false,
		DefaultLevel: "info",
		PassThrough:  true,
		Tags:         map[string]string{"tag": "tag"},
	}, NewEnvVar())
	assert.NoError(t, err)

	logger.Write("test")
	assert.Equal(t,
		`{"level":"info","tag":"tag","time":1579946400,"message":"test"}`+"\n",
		buffer.String())
}

func TestLog_NoStructuredConsoleFiltered(t *testing.T) {
	var buffer bytes.Buffer
	logger, err := NewLogger(context.Background(), StdOut, &buffer, "error", LogConfig{
		Structured:   false,
		DefaultLevel: "info",
		PassThrough:  true,
		Tags:         map[string]string{"tag": "tag"},
	}, NewEnvVar())
	assert.NoError(t, err)

	// default level is info, but logger accept only error
	logger.Write("test")
	assert.Equal(t, "", buffer.String())
}

func TestLog_StructuredConsole(t *testing.T) {
	var buffer bytes.Buffer
	logger, err := NewLogger(context.Background(), StdOut, &buffer, "info", LogConfig{
		Structured:   true,
		DefaultLevel: "info",
		PassThrough:  true,
		Tags:         map[string]string{"tag": "tag"},
	}, NewEnvVar())
	assert.NoError(t, err)

	// level field overwrite default logLevel
	logger.Write(`{"level": "error", "user": "shibukawa", "message": "error happens"}`)
	assert.Equal(t,
		`{"level":"error","tag":"tag","message":"error happens","user":"shibukawa","time":1579946400}`+"\n",
		buffer.String())
}

func TestLog_StructuredConsoleWithMask(t *testing.T) {
	var buffer bytes.Buffer
	logger, err := NewLogger(context.Background(), StdOut, &buffer, "info", LogConfig{
		Structured:   true,
		DefaultLevel: "info",
		PassThrough:  true,
		Mask:         []string{"password"},
	}, NewEnvVar())
	assert.NoError(t, err)

	// level field overwrite default logLevel
	logger.Write(`{"level": "error", "user": "shibukawa", "password": "should not show this"}`)
	assert.Equal(t,
		`{"level":"error","password":"********","user":"shibukawa","time":1579946400}`+"\n",
		buffer.String())
}

func TestLog_StructuredTransport(t *testing.T) {
	var buffer bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger, err := NewLogger(ctx, StdOut, &buffer, "info", LogConfig{
		Structured:   true,
		DefaultLevel: "info",
		PassThrough:  true,
		ExportConfig: "mem://stdout",
		Tags:         map[string]string{"tag": "tag"},
	}, NewEnvVar())
	assert.NoError(t, err)

	sub, err := pubsub.OpenSubscription(ctx, "mem://stdout")
	assert.NoError(t, err)

	// level field overwrite default logLevel
	logger.Write(`{"level": "error", "user": "shibukawa", "message": "error happens"}`)

	msg, err := sub.Receive(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "error", msg.Metadata["level"])
	assert.Equal(t, "error happens", msg.Metadata["message"])
	assert.Equal(t, "shibukawa", msg.Metadata["user"])
	assert.Equal(t, "tag", msg.Metadata["tag"])
}

func TestLog_WriteMetrics(t *testing.T) {
	var buffer bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger, err := NewLogger(ctx, StdOut, &buffer, "info", LogConfig{
		Structured:   true,
		DefaultLevel: "info",
		PassThrough:  true,
		ExportConfig: "mem://stdout",
		Tags:         map[string]string{"tag": "tag"},
	}, NewEnvVar())
	assert.NoError(t, err)

	sub, err := pubsub.OpenSubscription(ctx, "mem://stdout")
	assert.NoError(t, err)

	logger.WriteMetrics(10*1000*1000, 0.1, 0.1)

	assert.Equal(t,
		`{"level":"info","docradle-log":"metrics","mem-uasge":10000000,"mem-percent":0.1,"cpu-percent":0.1,"tag":"tag","time":1579946400}`+"\n",
		buffer.String())

	msg, err := sub.Receive(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "info", msg.Metadata["level"])
	assert.Equal(t, "metrics", msg.Metadata["docradle-log"])
	assert.Equal(t, "10000000", msg.Metadata["mem-usage"])
	assert.Equal(t, "0.1", msg.Metadata["mem-percent"])
	assert.Equal(t, "0.1", msg.Metadata["cpu-percent"])
}

func TestLog_WriteProcessStart(t *testing.T) {
	var buffer bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger, err := NewLogger(ctx, StdOut, &buffer, "info", LogConfig{
		Structured:   true,
		DefaultLevel: "info",
		PassThrough:  true,
		ExportConfig: "mem://stdout",
		Tags:         map[string]string{"tag": "tag"},
	}, NewEnvVar())
	assert.NoError(t, err)

	sub, err := pubsub.OpenSubscription(ctx, "mem://stdout")
	assert.NoError(t, err)

	startAt := time.Date(2020, time.January, 25, 10, 0, 0, 0, time.UTC)
	logger.WriteProcessStart(startAt, 10, "/home/root", "time", []string{"echo", "hello"})

	assert.Equal(t,
		`{"level":"info","docradle-log":"start","process-id":10,"work-directory":"/home/root","command":"time","arguments":["echo","hello"],"tag":"tag","time":1579946400}`+"\n",
		buffer.String())

	msg, err := sub.Receive(ctx)
	assert.NoError(t, err)

	assert.Equal(t, "info", msg.Metadata["level"])
	assert.Equal(t, "start", msg.Metadata["docradle-log"])
	assert.Equal(t, "time", msg.Metadata["command"])
	assert.Equal(t, "10", msg.Metadata["process-id"])
	assert.Equal(t, "/home/root", msg.Metadata["work-directory"])
	assert.Equal(t, "echo hello", msg.Metadata["arguments"])
	assert.Equal(t, "1579946400", msg.Metadata["time"])
}
