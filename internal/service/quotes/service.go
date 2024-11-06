package quotes

import (
	"context"

	"github.com/IskenT/spin-wisdom/internal/service/quotes/model"
	"github.com/IskenT/spin-wisdom/internal/service/quotes/repo"
)

type Service struct {
	repo repo.QuoteRepo
}

func NewService(repo repo.QuoteRepo) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) GetRandomQuote(ctx context.Context) model.Quote {
	return s.repo.GetRandomQuote(ctx)
}

func (s *Service) Cleanup(ctx context.Context) error {
	//ToDo: cleanup
	return nil
}
