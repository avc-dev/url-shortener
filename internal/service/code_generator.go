package service

import (
	"math/rand"

	"github.com/avc-dev/url-shortener/internal/model"
)

const (
	CodeLength   = 8
	AllowedChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// CodeGenerator реализует генератор кодов с использованием вероятностного подхода
type CodeGenerator struct {
	random *rand.Rand
}

// NewCodeGenerator создает новый генератор кодов
func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{
		random: rand.New(rand.NewSource(rand.Int63())),
	}
}

// GenerateCode генерирует случайный код
func (g *CodeGenerator) GenerateCode() model.Code {
	return model.Code(g.generateRandomString())
}

// GenerateBatchCodes генерирует указанное количество случайных кодов
func (g *CodeGenerator) GenerateBatchCodes(count int) []model.Code {
	codes := make([]model.Code, count)
	for i := 0; i < count; i++ {
		codes[i] = g.GenerateCode()
	}
	return codes
}

// generateRandomString генерирует случайную строку заданной длины
func (g *CodeGenerator) generateRandomString() string {
	result := make([]byte, CodeLength)

	for i := range result {
		result[i] = AllowedChars[g.random.Intn(len(AllowedChars))]
	}

	return string(result)
}
