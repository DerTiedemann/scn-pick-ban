package lcu

import "github.com/pkg/errors"

type Options struct {
	Token    string
	Protocol string
	Port     uint16
}

func (o *Options) validate() error {
	if !(o.Protocol == "http" || o.Protocol == "https") {
		return errors.Errorf("%s is not a valid protocol", o.Protocol)
	}

	return nil
}

type Message struct {
	Data      map[string]interface{} `mapstructure:"data"`
	EventType string                 `mapstructure:"eventType"`
	Uri       string                 `mapstructure:"uri"`
}
