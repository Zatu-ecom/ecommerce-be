package singleton

import (
	"ecommerce-be/common/db"
	"ecommerce-be/report/repository"
)

type RepositoryFactory struct {
	reportRepository repository.ReportRepository
}

func NewRepositoryFactory() *RepositoryFactory {
	return &RepositoryFactory{
		reportRepository: repository.NewReportRepository(db.GetDB()),
	}
}

func (f *RepositoryFactory) GetReportRepository() repository.ReportRepository {
	return f.reportRepository
}
