package service

import (
	"fmt"
	"math/rand"
)

const (
	CodeLength   = 8
	MaxTries     = 100
	AllowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func randomString() string {
	result := make([]byte, CodeLength)

	for i := range result {
		result[i] = AllowedChars[rand.Intn(len(AllowedChars))]
	}

	return string(result)
}

func GenerateCode(checkerFunc func(code string) error) (string, error) {
	for tries := 0; tries < MaxTries; tries++ {
		uniqueCode := randomString()

		err := checkerFunc(uniqueCode)
		if err == nil {
			return uniqueCode, nil
		}
	}

	return "", fmt.Errorf("could not generate unique code")
}
