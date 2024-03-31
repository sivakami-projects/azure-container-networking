package log

import (
	"fmt"
	"os"
	"runtime"

	"github.com/Azure/azure-container-networking/cni/log/ETWZapCore"
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
	etwCNIEventName    = "Azure-CNI"
)

func initZapLog(logFile string, isEtwLoggingEnabled bool) *zap.Logger {
	logFileCNIWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   LogPath + logFile,
		MaxSize:    maxLogFileSizeInMb,
		MaxBackups: maxLogFileCount,
	})

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	textfilecore := zapcore.NewCore(jsonEncoder, logFileCNIWriter, zapcore.DebugLevel)
	Logger := zap.New(textfilecore, zap.AddCaller())

	if isEtwLoggingEnabled && runtime.GOOS == "windows" {
		etwSyncer, err := ETWZapCore.NewEtwWriteSyncer(etwCNIEventName)
		if err != nil {
			fmt.Printf("Failed to initialize ETW logger: %v. Defaulting to standard logger.\n", err)
		} else {
			etwcore := zapcore.NewCore(jsonEncoder, zapcore.AddSync(etwSyncer), zap.InfoLevel)
			teecore := zapcore.NewTee(textfilecore, etwcore)
			Logger = zap.New(teecore, zap.AddCaller())
		}
	}
	return Logger.With(zap.Int("pid", os.Getpid()))
}

var (
	CNILogger       = initZapLog(zapCNILogFile, true)
	IPamLogger      = initZapLog(zapIpamLogFile, true)
	TelemetryLogger = initZapLog(zapTelemetryLogFile, true)
)
