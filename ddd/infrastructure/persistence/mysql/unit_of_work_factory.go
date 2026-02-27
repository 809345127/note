package mysql

import (
	"ddd/domain/shared"
	"ddd/infrastructure/persistence/retry"

	"gorm.io/gorm"
)

type UnitOfWorkFactory struct {
	db          *gorm.DB
	retryConfig retry.Config
}

func NewUnitOfWorkFactory(db *gorm.DB, retryConfig retry.Config) *UnitOfWorkFactory {
	return &UnitOfWorkFactory{
		db:          db,
		retryConfig: retryConfig,
	}
}
func (f *UnitOfWorkFactory) New() shared.UnitOfWork {
	uow := NewUnitOfWork(f.db)
	uow.SetRetryConfig(f.retryConfig)
	return uow
}

var _ shared.UnitOfWorkFactory = (*UnitOfWorkFactory)(nil)
