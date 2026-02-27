package shared

import "context"

type Specification[T any] interface {
	IsSatisfiedBy(ctx context.Context, entity T) bool
}

type AndSpecification[T any] struct {
	Left  Specification[T]
	Right Specification[T]
}

func (spec AndSpecification[T]) IsSatisfiedBy(ctx context.Context, entity T) bool {
	return spec.Left.IsSatisfiedBy(ctx, entity) && spec.Right.IsSatisfiedBy(ctx, entity)
}

func And[T any](left, right Specification[T]) Specification[T] {
	return AndSpecification[T]{Left: left, Right: right}
}

type OrSpecification[T any] struct {
	Left  Specification[T]
	Right Specification[T]
}

func (spec OrSpecification[T]) IsSatisfiedBy(ctx context.Context, entity T) bool {
	return spec.Left.IsSatisfiedBy(ctx, entity) || spec.Right.IsSatisfiedBy(ctx, entity)
}

func Or[T any](left, right Specification[T]) Specification[T] {
	return OrSpecification[T]{Left: left, Right: right}
}

type NotSpecification[T any] struct {
	Spec Specification[T]
}

func (spec NotSpecification[T]) IsSatisfiedBy(ctx context.Context, entity T) bool {
	return !spec.Spec.IsSatisfiedBy(ctx, entity)
}

func Not[T any](inner Specification[T]) Specification[T] {
	return NotSpecification[T]{Spec: inner}
}
