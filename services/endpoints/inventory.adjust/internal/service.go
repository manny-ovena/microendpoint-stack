package internal

import "errors"

var ErrNotImplemented = errors.New("inventory adjustment is not implemented")

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) Adjust(req AdjustRequest) (AdjustResponse, error) {
	if err := ValidateAdjustRequest(req); err != nil {
		return AdjustResponse{}, err
	}

	return AdjustResponse{}, ErrNotImplemented
}
