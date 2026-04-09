// SPDX-License-Identifier: BUSL-1.1

package alerts

import (
	"encoding/json"
	"fmt"
)

// AlerterFactory is a constructor function that creates an Alerter from raw JSON config.
type AlerterFactory func(config json.RawMessage) (Alerter, error)

var registry = map[string]AlerterFactory{}

// RegisterAlerter registers a factory for the given channel type.
// It is typically called from init() functions in channel implementation files.
func RegisterAlerter(channelType string, factory AlerterFactory) {
	registry[channelType] = factory
}

// NewAlerter creates an Alerter for the given channel type using the registered factory.
// Returns an error if the channel type has not been registered.
func NewAlerter(channelType string, config json.RawMessage) (Alerter, error) {
	factory, ok := registry[channelType]
	if !ok {
		return nil, fmt.Errorf("unknown alert channel type: %s", channelType)
	}
	return factory(config)
}

// IsRegisteredType reports whether a factory has been registered for the given channel type.
func IsRegisteredType(channelType string) bool {
	_, ok := registry[channelType]
	return ok
}
