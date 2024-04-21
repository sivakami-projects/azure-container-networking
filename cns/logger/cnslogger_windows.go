package logger

import (
	"github.com/Azure/azure-container-networking/zapetw"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	// LogPath is the path where log files are stored.
	LogPath         = ""
	etwCNSEventName = "AzureCNS"
)

func GetPlatformCores(loggingLevel zapcore.Level) (zapcore.Core, error) {
	etwcore, err := GetETWCore(loggingLevel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ETW core.")
	}
	return etwcore, nil
}

func GetETWCore(loggingLevel zapcore.Level) (zapcore.Core, error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

	etwcore, err := zapetw.NewETWCore(etwCNSEventName, jsonEncoder, loggingLevel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ETW core")
	}
	return etwcore, nil
}
