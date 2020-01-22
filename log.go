package docradle

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/future-architect/fluentdpub"
	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/kafkapubsub"
	_ "gocloud.dev/pubsub/mempubsub"
	"golang.org/x/sync/errgroup"
)

type LogType int

func (l LogType) String() string {
	switch l {
	case StdOut:
		return "stdout"
	case StdErr:
		return "stderr"
	}
	return "Unknown LogType"
}

const (
	StdOut LogType = iota + 1
	StdErr

	LogLevelKey       = "level"
	LogDocradleLogKey = "docradle-log"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
}

type Logger struct {
	console      *zerolog.Logger
	transporter  *pubsub.Topic
	defaultLevel zerolog.Level
	logLevel     zerolog.Level
	tags         map[string]string
	maskKeys     []string
	structured   bool
}

var logLevelMap = map[string]zerolog.Level{
	"trace": zerolog.TraceLevel,
	"debug": zerolog.DebugLevel,
	"info":  zerolog.InfoLevel,
	"warn":  zerolog.WarnLevel,
	"error": zerolog.ErrorLevel,
}

func NewLogger(ctx context.Context, logType LogType, writer io.Writer, logLevelLabel string, logConfig LogConfig, envvar *EnvVar) (*Logger, error) {
	var transporter *pubsub.Topic
	if logConfig.ExportConfig != "" {
		u, err := url.Parse(envvar.Expand(logConfig.ExportConfig))
		if err != nil {
			return nil, fmt.Errorf("Can't parse")
		}
		switch u.Scheme {
		case "fluentd":
			transporter, err = fluentdpub.OpenTopicURL(ctx, u, envvar.Expand(logConfig.ExportHost))
			if err != nil {
				return nil, fmt.Errorf("Can't init fluentd exporter for %s: %w", logType.String(), err)
			}
		case "kafka":
			addrs := []string{envvar.Expand(logConfig.ExportHost)}
			config := kafkapubsub.MinimalConfig()
			transporter, err = kafkapubsub.OpenTopic(addrs, config, u.Hostname(), nil)
			if err != nil {
				return nil, fmt.Errorf("Can't init kafka exporter for %s: %w", logType.String(), err)
			}
		case "mem":
			transporter, err = pubsub.OpenTopic(ctx, envvar.Expand(logConfig.ExportConfig))
			if err != nil {
				return nil, fmt.Errorf("Can't init mem exporter for %s: %w", logType.String(), err)
			}
		}
	}
	logLevel := logLevelMap[logLevelLabel]

	logger := &Logger{
		defaultLevel: logLevelMap[logConfig.DefaultLevel],
		logLevel:     logLevel,
		transporter:  transporter,
		structured:   logConfig.Structured,
		tags:         logConfig.Tags,
		maskKeys:     logConfig.Mask,
	}

	if logConfig.PassThrough {
		isTerminal := false
		if f, ok := writer.(*os.File); ok {
			if isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd()) {
				isTerminal = true
			}
		}
		if isTerminal {
			console := log.Output(zerolog.ConsoleWriter{Out: writer}).Level(logLevel)
			logger.console = &console
		} else {
			console := log.Output(writer).Level(logLevel)
			logger.console = &console
		}
	}

	return logger, nil
}

func (l *Logger) StartOutput(eg *errgroup.Group, reader io.ReadCloser) {
	eg.Go(func() error {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			l.Write(scanner.Text())
		}
		return nil
	})
}

func (l *Logger) Write(line string) {
	jsonMap := make(map[string]interface{})
	parsed := false
	if l.structured {
		if err := json.Unmarshal([]byte(line), &jsonMap); err == nil {
			parsed = true
		}
	}
	if parsed {
		l.WriteMap(jsonMap)
	} else {
		if l.console != nil {
			event := l.console.WithLevel(l.defaultLevel)
			for key, value := range l.tags {
				event.Str(key, value)
			}
			event.Msg(line)
		}
		if l.transporter != nil && l.defaultLevel >= l.logLevel {
			metadata := make(map[string]string, len(l.tags)+2)
			for key, value := range l.tags {
				metadata[key] = value
			}
			metadata["message"] = line
			metadata[LogLevelKey] = l.defaultLevel.String()

			l.transporter.Send(context.TODO(), &pubsub.Message{
				Metadata: metadata,
			})
		}
	}
}

func (l *Logger) WriteMap(log map[string]interface{}) {
	logLevel := l.logLevel
	if levelItem, ok := log[LogLevelKey]; ok {
		if levelLabel, ok := levelItem.(string); ok {
			if level, ok := logLevelMap[levelLabel]; ok {
				logLevel = level
			}
		}
		delete(log, LogLevelKey)
	}
	for _, maskKey := range l.maskKeys {
		if _, ok := log[maskKey]; ok {
			log[maskKey] = "********"
		}
	}
	if l.console != nil {
		event := l.console.WithLevel(logLevel)
		for key, value := range l.tags {
			if _, ok := log[key]; !ok {
				event.Str(key, value)
			}
		}
		event.Fields(log).Send()
	}
	if l.transporter != nil {
		metadata := make(map[string]string, len(log)+len(l.tags)+1)
		metadata[LogLevelKey] = logLevel.String()
		for key, value := range l.tags {
			metadata[key] = value
		}
		for key, value := range log {
			if svalue, ok := value.(string); ok {
				metadata[key] = svalue
			} else {
				metadata[key] = fmt.Sprintf("%v", value)
			}
		}
		l.transporter.Send(context.TODO(), &pubsub.Message{
			Metadata: metadata,
		})
	}
}

func (l *Logger) WriteMetrics(memUsage uint64, memPercent float32, cpuPercent float64) {
	if l.console != nil {
		event := l.console.WithLevel(zerolog.InfoLevel)
		event.Str(LogDocradleLogKey, "metrics").
			Uint64("mem-uasge", memUsage).
			Float32("mem-percent", memPercent).
			Float64("cpu-percent", cpuPercent)
		for key, value := range l.tags {
			event.Str(key, value)
		}
		event.Send()
	}
	if l.transporter != nil {
		metadata := make(map[string]string, len(l.tags)+5)
		metadata[LogLevelKey] = zerolog.InfoLevel.String()
		for key, value := range l.tags {
			metadata[key] = value
		}
		metadata[LogDocradleLogKey] = "metrics"
		metadata["mem-usage"] = strconv.FormatUint(memUsage, 10)
		metadata["mem-percent"] = strconv.FormatFloat(float64(memPercent), 'f', -1, 32)
		metadata["cpu-percent"] = strconv.FormatFloat(cpuPercent, 'f', -1, 32)
		l.transporter.Send(context.TODO(), &pubsub.Message{
			Metadata: metadata,
		})
	}
}

func (l *Logger) WriteProcessStart(startAt time.Time, pid int, dir, cmd string, args []string) {
	if l.console != nil {
		event := l.console.WithLevel(zerolog.InfoLevel)
		event.Str(LogDocradleLogKey, "start").
			Int("process-id", pid).
			Str("work-directory", dir).
			Str("command", cmd).
			Strs("arguments", args)
		for key, value := range l.tags {
			event.Str(key, value)
		}
		event.Send()
	}
	if l.transporter != nil {
		metadata := make(map[string]string, len(l.tags)+6)
		metadata[LogLevelKey] = zerolog.InfoLevel.String()
		for key, value := range l.tags {
			metadata[key] = value
		}
		metadata[LogDocradleLogKey] = "start"
		metadata["time"] = strconv.FormatInt(startAt.Unix(), 10)
		metadata["process-id"] = strconv.FormatInt(int64(pid), 10)
		metadata["work-directory"] = dir
		metadata["command"] = cmd
		metadata["arguments"] = strings.Join(args, " ")
		l.transporter.Send(context.TODO(), &pubsub.Message{
			Metadata: metadata,
		})
	}
}

func (l *Logger) WriteProcessResult(exitAt time.Time, status string, wallClock, user, sys time.Duration) {
	if l.console != nil {
		event := l.console.WithLevel(zerolog.InfoLevel)
		event.Str(LogDocradleLogKey, "result").
			Str("process-status", status).
			Dur("wallclock-time", wallClock).
			Dur("user-time", user).
			Dur("sysetm-time", sys)
		for key, value := range l.tags {
			event.Str(key, value)
		}
		event.Send()
	}
	if l.transporter != nil {
		metadata := make(map[string]string, len(l.tags)+7)
		metadata[LogLevelKey] = zerolog.InfoLevel.String()
		for key, value := range l.tags {
			metadata[key] = value
		}
		metadata[LogDocradleLogKey] = "result"
		metadata["time"] = strconv.FormatInt(exitAt.Unix(), 10)
		metadata["process-status"] = status
		metadata["wallclock-time"] = wallClock.String()
		metadata["user-time"] = user.String()
		metadata["system-time"] = sys.String()
		l.transporter.Send(context.TODO(), &pubsub.Message{
			Metadata: metadata,
		})
	}
}

func (l *Logger) Close() {
	if l.transporter != nil {
		l.transporter.Shutdown(context.TODO())
	}
}
