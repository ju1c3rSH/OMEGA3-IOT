package repository

import (
	"gorm.io/gorm"
)

// UnitOfWork manages transactions across multiple repositories
type UnitOfWork struct {
	db         *gorm.DB
	tx         *gorm.DB
	committed  bool
}

// NewUnitOfWork creates a new unit of work
func NewUnitOfWork(db *gorm.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

// Begin starts a transaction
func (u *UnitOfWork) Begin() *UnitOfWork {
	u.tx = u.db.Begin()
	return u
}

// Commit commits the transaction
func (u *UnitOfWork) Commit() error {
	if u.tx == nil {
		return nil
	}
	if err := u.tx.Commit().Error; err != nil {
		return err
	}
	u.committed = true
	return nil
}

// Rollback rolls back the transaction if not committed
func (u *UnitOfWork) Rollback() {
	if u.tx != nil && !u.committed {
		u.tx.Rollback()
	}
}

// UserRepository returns a UserRepository with the current transaction
func (u *UnitOfWork) UserRepository() UserRepository {
	if u.tx != nil {
		return NewUserRepository(u.tx)
	}
	return NewUserRepository(u.db)
}

// InstanceRepository returns an InstanceRepository with the current transaction
func (u *UnitOfWork) InstanceRepository() InstanceRepository {
	if u.tx != nil {
		return NewInstanceRepository(u.tx)
	}
	return NewInstanceRepository(u.db)
}

// DeviceRegistrationRecordRepository returns a DeviceRegistrationRecordRepository with the current transaction
func (u *UnitOfWork) DeviceRegistrationRecordRepository() DeviceRegistrationRecordRepository {
	if u.tx != nil {
		return NewDeviceRegistrationRecordRepository(u.tx)
	}
	return NewDeviceRegistrationRecordRepository(u.db)
}

// DeviceShareRepository returns a DeviceShareRepository with the current transaction
func (u *UnitOfWork) DeviceShareRepository() DeviceShareRepository {
	if u.tx != nil {
		return NewDeviceShareRepository(u.tx)
	}
	return NewDeviceShareRepository(u.db)
}

// ExecuteInTransaction executes a function within a transaction
func ExecuteInTransaction(db *gorm.DB, fn func(uow *UnitOfWork) error) error {
	uow := NewUnitOfWork(db).Begin()
	defer uow.Rollback()

	if err := fn(uow); err != nil {
		return err
	}

	return uow.Commit()
}
