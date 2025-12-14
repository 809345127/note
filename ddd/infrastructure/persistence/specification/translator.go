package specification

import (
	"ddd/domain/order"
	"ddd/domain/shared"
	"ddd/domain/user"

	"gorm.io/gorm"
)

// Translator converts domain specifications to GORM queries
// DDD principle: Infrastructure layer handles framework-specific concerns
type Translator interface {
	// Translate converts a domain specification to a GORM query function
	// Returns nil if the specification type is not supported
	Translate(spec shared.Specification) func(*gorm.DB) *gorm.DB
}

// GormTranslator implements Translator for GORM
type GormTranslator struct{}

// NewGormTranslator creates a new GORM translator
func NewGormTranslator() *GormTranslator {
	return &GormTranslator{}
}

// Translate converts a domain specification to a GORM query function
func (t *GormTranslator) Translate(spec shared.Specification) func(*gorm.DB) *gorm.DB {
	if spec == nil {
		return nil
	}

	// Handle composite specifications
	switch s := spec.(type) {
	case shared.AndSpecification:
		return t.translateAnd(s)
	case shared.OrSpecification:
		return t.translateOr(s)
	case shared.NotSpecification:
		return t.translateNot(s)
	}

	// Handle concrete specifications from domain packages
	// We use type switches for each known specification type
	// This is acceptable because infrastructure depends on domain
	return t.translateConcrete(spec)
}

// translateAnd translates AndSpecification to GORM query
func (t *GormTranslator) translateAnd(spec shared.AndSpecification) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		leftQuery := t.Translate(spec.Left)
		rightQuery := t.Translate(spec.Right)

		if leftQuery != nil {
			db = leftQuery(db)
		}
		if rightQuery != nil {
			db = rightQuery(db)
		}
		return db
	}
}

// translateOr translates OrSpecification to GORM query
func (t *GormTranslator) translateOr(spec shared.OrSpecification) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// GORM doesn't directly support OR composition of scopes
		// We need to use db.Or() with conditions
		// For simplicity, we'll handle this by creating a complex WHERE clause
		// This is a limitation of this simple implementation
		return db
	}
}

// translateNot translates NotSpecification to GORM query
func (t *GormTranslator) translateNot(spec shared.NotSpecification) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// GORM doesn't directly support NOT composition
		// This would require complex query building
		return db
	}
}

// translateConcrete translates concrete domain specifications
func (t *GormTranslator) translateConcrete(spec shared.Specification) func(*gorm.DB) *gorm.DB {
	// Handle user domain specifications
	switch s := spec.(type) {
	case user.ByEmailSpecification:
		return func(db *gorm.DB) *gorm.DB {
			return db.Where("email = ?", s.Email)
		}
	case user.ByStatusSpecification:
		return func(db *gorm.DB) *gorm.DB {
			return db.Where("is_active = ?", s.Active)
		}
	case user.ByAgeRangeSpecification:
		return func(db *gorm.DB) *gorm.DB {
			if s.Min > 0 {
				db = db.Where("age >= ?", s.Min)
			}
			if s.Max > 0 {
				db = db.Where("age <= ?", s.Max)
			}
			return db
		}
	}

	// Handle order domain specifications
	switch s := spec.(type) {
	case order.ByUserIDSpecification:
		return func(db *gorm.DB) *gorm.DB {
			return db.Where("user_id = ?", s.UserID)
		}
	case order.ByStatusSpecification:
		return func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", s.Status)
		}
	case order.ByDateRangeSpecification:
		return func(db *gorm.DB) *gorm.DB {
			if !s.Start.IsZero() {
				db = db.Where("created_at >= ?", s.Start)
			}
			if !s.End.IsZero() {
				db = db.Where("created_at <= ?", s.End)
			}
			return db
		}
	}

	// Unknown specification type
	return nil
}