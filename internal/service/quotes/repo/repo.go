package repo

import (
	"context"

	"github.com/IskenT/spin-wisdom/internal/service/quotes/model"
)

type QuoteRepo interface {
	GetRandomQuote(ctx context.Context) model.Quote
}
