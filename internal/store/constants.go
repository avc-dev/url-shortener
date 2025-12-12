package store

import "math/rand"

const (
	CodeLength   = 8
	AllowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func randomString() string {
	result := make([]byte, CodeLength)

	for i := range result {
		result[i] = AllowedChars[rand.Intn(len(AllowedChars))]
	}

	return string(result)
}
