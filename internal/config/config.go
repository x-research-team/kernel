package config

import (
	"encoding/json"

	"github.com/x-research-team/bus"
	"github.com/x-research-team/utils/file"
)

type TLogLevelType string

type TLogLevelTypes []TLogLevelType

func (levels *TLogLevelTypes) Has(level *TLogLevelType) bool {
	for _, l := range *levels {
		if l == *level {
			return true
		}
	}
	return false
}

var LogLevels = func() *TLogLevelTypes {
	v := new(TLogLevelTypes)
	if err := file.Read("config", "log.json", v); err != nil {
		panic(err)
	}
	return v
}()

var Extensions = func() *TExtensionTypes {
	v := new(TExtensionTypes)
	if err := file.Read("config", "extensions.json", v); err != nil {
		panic(err)
	}
	return v
}()

type TLogLevel map[TLogLevelType]bool
type TLogConfig struct {
	Level TLogLevel `json:"level"`
}

func (log TLogLevel) ToJson() json.RawMessage {
	buffer, err := json.Marshal(log)
	if err != nil {
		bus.Error <- err
	}
	return buffer
}

type TExtensionType string

type TExtensionTypes []TExtensionType

type TExtensionConfig struct {
	Path    string          `json:"path"`
	Enabled bool            `json:"enabled"`
	Types   TExtensionTypes `json:"types"`
}

type TExtensionConfigs []TExtensionConfig

type TConfig struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Log        *TLogConfig       `json:"log"`
	Extensions TExtensionConfigs `json:"extensions"`
}

var Config = func() *TConfig {
	v := new(TConfig)
	if err := file.Read("config", "kernel.json", v); err != nil {
		panic(err)
	}
	v.Log = new(TLogConfig)
	v.Log.Level = make(TLogLevel)
	for _, level := range *LogLevels {
		v.Log.Level[level] = true
	}
	for i := range v.Extensions {
		if v.Extensions[i].Types == nil {
			v.Extensions[i].Types = *Extensions
		}
	}
	return v
}()
