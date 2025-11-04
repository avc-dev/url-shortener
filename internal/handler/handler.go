package handler

import (
	"github.com/avc-dev/url-shortener/internal/model"
)

type URLRepository interface {
	CreateURL(code model.Code, url model.URL) error
	GetURLByCode(code model.Code) (model.URL, error)
}

type Usecase struct {
	repo URLRepository
}

func New(repo URLRepository) *Usecase {
	return &Usecase{repo}
}
