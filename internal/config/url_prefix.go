package config

import (
	"fmt"
	"strings"
)

type URLPrefix string

func (p URLPrefix) String() string {
	return string(p)
}

func (p *URLPrefix) Set(value string) error {
	if !strings.HasPrefix(value, "http") {
		return fmt.Errorf("invalid URL prefix format: %s", value)
	}

	*p = URLPrefix(value)

	return nil
}

func (p *URLPrefix) UnmarshalText(text []byte) error {
	return p.Set(string(text))
}
