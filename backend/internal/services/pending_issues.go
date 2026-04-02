package services

import "github.com/hao/game/internal/domain"

type PendingIssuesService struct{}

func NewPendingIssuesService() *PendingIssuesService {
	return &PendingIssuesService{}
}

func (s *PendingIssuesService) Catalog() domain.PendingIssueCatalog {
	return domain.PendingIssueCatalogDefinitions()
}
