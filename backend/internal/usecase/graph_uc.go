package usecase

import (
	"backend/internal/domain"
	"context"
)

type graphUsecase struct {
	BaseUsecase
}

func NewGraphUsecase(base BaseUsecase) domain.GraphUsecase {
	return &graphUsecase{BaseUsecase: base}
}

func (uc *graphUsecase) GetGraph(ctx context.Context, input string) ([]domain.CytoElement, error) {
	isTxHash := len(input) == 66
	return uc.TxRepo.GetGraph(ctx, input, isTxHash)
}