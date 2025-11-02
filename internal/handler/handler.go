package handler

import (
	"github.com/avc-dev/url-shortener/internal/repository"
)

type Usecase struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Usecase {
	return &Usecase{repo}
}
