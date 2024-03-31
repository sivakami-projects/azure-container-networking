//go:build windows
// +build windows

package ETWZapCore

import (
	"fmt"

	"github.com/Microsoft/go-winio/pkg/etw"
	"github.com/Microsoft/go-winio/pkg/guid"
)

type EtwWriteSyncer struct {
	provider  *etw.Provider
	eventName string
}

func etwEventCallback(sourceID guid.GUID, state etw.ProviderState, level etw.Level, matchAnyKeyword uint64, matchAllKeyword uint64, filterData uintptr) {
	fmt.Printf("ETW Callback: isEnabled=%d, level=%d, matchAnyKeyword=%d\n", state, level, matchAnyKeyword)
}

func NewEtwWriteSyncer(eventName string) (*EtwWriteSyncer, error) {

	// GUID, err := guid.FromString("")
	// if err != nil {
	// 	fmt.Println(err)
	// }

	provider, err := etw.NewProviderWithOptions("azure-container-networking-ccp", etw.WithCallback(etwEventCallback))
	if err != nil {
		return nil, err
	}

	return &EtwWriteSyncer{
		provider:  provider,
		eventName: eventName,
	}, nil
}

func (e *EtwWriteSyncer) Write(p []byte) (int, error) {

	err := e.provider.WriteEvent(
		e.eventName,
		etw.WithEventOpts(
			etw.WithLevel(etw.LevelAlways),
			etw.WithKeyword(0x1),
		),
		[]etw.FieldOpt{
			etw.StringField("Message", string(p)),
		},
	)

	if err != nil {
		return 0, err
	}
	return len(p), nil

}

// flush any buffered data to the underlying log destination,
// ensuring that all logged data is actually written out and not just held in memory.
func (e *EtwWriteSyncer) Sync() error {
	return nil
}
