package config

import (
	"flag"
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
		return err
	}
	a.Port = port

	return nil
}

type URLPrefix string

func (b URLPrefix) String() string {
	return string(b)
}

func (b *URLPrefix) Set(value string) error {
	if !strings.HasPrefix(value, "http") {
		return fmt.Errorf("invalid URL prefix format: %s", value)
	}

	value = strings.TrimSuffix(value, "/")
	value = value + "/"

	*b = URLPrefix(value)

	return nil
}

var (
	Address NetworkAddress = NetworkAddress{Host: "localhost", Port: 8000}
	BaseURL URLPrefix      = URLPrefix("http://localhost:8000")
)

func init() {
	flag.Var(&Address, "a", "address to run HTTP server")
	flag.Var(&BaseURL, "b", "base URL for shortened URL")
}
