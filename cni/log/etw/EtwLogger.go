package etw

import (
	"fmt"

	"github.com/Microsoft/go-winio/pkg/etw"
	"github.com/Microsoft/go-winio/pkg/guid"
	"go.uber.org/zap"
)

func etwEventCallback(sourceID guid.GUID, state etw.ProviderState, level etw.Level, matchAnyKeyword uint64, matchAllKeyword uint64, filterData uintptr) {
	fmt.Printf("ETW Callback: isEnabled=%d, level=%d, matchAnyKeyword=%d\n", state, level, matchAnyKeyword)
}

// Provider is an interface that represents an ETW logging provider.
type Provider interface {
	WriteEvent(name string, eventOpts []etw.EventOpt, fieldOpts []etw.FieldOpt) error
}

// Logger is a type that allows for Event Tracing on Windows (ETW).
type Logger struct {
	etwProvider Provider
	etwDims     []etw.FieldOpt
}

func newLoggerInternal(dims []zap.Field, ep Provider) (*Logger, error) {
	logger := &Logger{etwProvider: ep}

	for _, dim := range dims {
		logger.etwDims = append(logger.etwDims, etw.StringField(dim.Key, dim.String))
	}

	return logger, nil
}

// NewLogger creates a new Logger that writes ETW events.
func NewLogger(dims []zap.Field) (*Logger, error) {

	// GUID, err := guid.FromString("f1d3d5f0-212c-50fa-c3f1-d54060252903")
	// if err != nil {
	// 	fmt.Println(err)
	// }

	provider, err := etw.NewProviderWithOptions("AzureCNI", etw.WithCallback(etwEventCallback))
	if err != nil {
		return nil, err
	}
	return newLoggerInternal(dims, provider)
}

// LogToEtw logs an ETW event at the specified level.
func (l *Logger) LogToEtw(level Level, msg string) {
	var loggingLevel etw.Level
	switch level {
	case Debug:
		loggingLevel = etw.LevelInfo
	case Error:
		loggingLevel = etw.LevelError
	}
	l.etwProvider.WriteEvent(
		"AzureCNIEvents",
		etw.WithEventOpts(
			etw.WithLevel(loggingLevel),
			etw.WithKeyword(0x1),
		),
		append(l.etwDims, etw.StringField("Message", msg)),
	)
}
