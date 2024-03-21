package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	zapCNILogFile       = "azure-vnet.log"
	zapIpamLogFile      = "azure-vnet-ipam.log"
	zapTelemetryLogFile = "azure-vnet-telemetry.log"
)

const (
	maxLogFileSizeInMb = 5
	maxLogFileCount    = 8
)

type etwLogger interface {
	LogToEtw(level string, msg string)
}

type cniLogger struct {
	ZapLogger    *zap.Logger
	EtwLogger    etwLogger
	IsETWEnabled bool
}

func initZapLog(logFile string) *zap.Logger {
	logFileCNIWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   LogPath + logFile,
		MaxSize:    maxLogFileSizeInMb,
		MaxBackups: maxLogFileCount,
	})

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	core := zapcore.NewCore(jsonEncoder, logFileCNIWriter, zapcore.DebugLevel)
	Logger := zap.New(core)
	return Logger.With(zap.Int("pid", os.Getpid()))
}

var (
	CNILogger       = &cniLogger{ZapLogger: initZapLog(zapCNILogFile)}
	IPamLogger      = &cniLogger{ZapLogger: initZapLog(zapIpamLogFile)}
	TelemetryLogger = &cniLogger{ZapLogger: initZapLog(zapTelemetryLogFile)}
)

func (l *cniLogger) With(fields ...zap.Field) *cniLogger {
	return &cniLogger{
		ZapLogger:    l.ZapLogger.With(fields...),
		EtwLogger:    l.EtwLogger,
		IsETWEnabled: l.IsETWEnabled,
	}
}

func (l *cniLogger) Info(msg string, fields ...zap.Field) {
	l.ZapLogger.Info(msg, fields...)
	if l.IsETWEnabled && l.EtwLogger != nil {
		l.EtwLogger.LogToEtw("INFO", msg)
	}
}

func (l *cniLogger) Debug(msg string, fields ...zap.Field) {
	l.ZapLogger.Debug(msg, fields...)
	if l.IsETWEnabled && l.EtwLogger != nil {
		l.EtwLogger.LogToEtw("DEBUG", msg)
	}
}

func (l *cniLogger) Warn(msg string, fields ...zap.Field) {
	l.ZapLogger.Warn(msg, fields...)
	if l.IsETWEnabled && l.EtwLogger != nil {
		l.EtwLogger.LogToEtw("WARN", msg)
	}
}

func (l *cniLogger) Error(msg string, fields ...zap.Field) {
	l.ZapLogger.Error(msg, fields...)
	if l.IsETWEnabled && l.EtwLogger != nil {
		// rewrite the message to include the fields.
		l.EtwLogger.LogToEtw("ERROR", msg)
	}
}
