package handler

import (
	"github.com/avc-dev/url-shortener/internal/model"
)

type URLRepository interface {
	CreateURL(code model.Code, url model.URL) error
	GetURLByCode(code model.Code) (model.URL, error)
}

type URLService interface {
	CreateShortURL(originalURL model.URL) (model.Code, error)
}

type Usecase struct {
	repo    URLRepository
	service URLService
}

func New(repo URLRepository, service URLService) *Usecase {
	return &Usecase{
		repo:    repo,
		service: service,
	}
}
