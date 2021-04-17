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

var ComponentConfigs = func() TComponentConfigs {
	v := make(TComponentConfigs, 0)
	p := make(TComponentPaths, 0)
	if err := file.Read("config", "components.json", &p); err != nil {
		panic(err)
	}
	types := ComponentTypes
	for i := range p {
		v = append(v, TComponentConfig{
			Types:   types,
			Path:    p[i],
			Enabled: true,
		})
	}
	return v
}()

var ComponentTypes = func() TComponentTypes {
	v := make(TComponentTypes, 0)
	if err := file.Read("config", "extensions.json", &v); err != nil {
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

type TComponentType string

type TComponentTypes []TComponentType

type TComponentPath string

type TComponentPaths []TComponentPath

type TComponentConfig struct {
	Path    TComponentPath  `json:"path"`
	Enabled bool            `json:"enabled"`
	Types   TComponentTypes `json:"types,omitempty"`
}

type TComponentConfigs []TComponentConfig

func (c TComponentConfigs) Paths() []string {
	paths := make([]string, 0)
	for _, config := range c {
		paths = append(paths, string(config.Path))
	}
	return paths
}

type TKernelConfig struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Log        *TLogConfig       `json:"log,omitempty"`
	Components TComponentConfigs `json:"components,omitempty"`
}

var Kernel = func() *TKernelConfig {
	v := new(TKernelConfig)
	if err := file.Read("config", "kernel.json", v); err != nil {
		panic(err)
	}
	v.Log = new(TLogConfig)
	v.Log.Level = make(TLogLevel)
	for _, level := range *LogLevels {
		v.Log.Level[level] = true
	}
	if v.Components == nil {
		v.Components = ComponentConfigs
	}
	return v
}()
