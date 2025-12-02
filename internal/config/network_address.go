package config

import (
	"fmt"
	"strconv"
	"strings"
)

type NetworkAddress struct {
	Host string
	Port int
}

func (a NetworkAddress) String() string {
	return a.Host + ":" + strconv.Itoa(a.Port)
}

func (a *NetworkAddress) Set(value string) error {
	parts := strings.Split(value, ":")

	if len(parts) != 2 {
		return fmt.Errorf("invalid network address format: %s", value)
	}

	a.Host = parts[0]

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}
	a.Port = port

	return nil
}

func (a *NetworkAddress) UnmarshalText(text []byte) error {
	return a.Set(string(text))
}
