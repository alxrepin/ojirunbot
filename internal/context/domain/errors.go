package domain

import "errors"

var (
	ErrNotFound = errors.New("not found")

	ErrUserNotFound    = wrapNotFound("user")
	ErrProfileNotFound = wrapNotFound("profile")
	ErrMealNotFound    = wrapNotFound("meal entry")
	ErrSessionNotFound = wrapNotFound("input session")
	ErrPhotoNotFound   = wrapNotFound("photo")

	ErrDailyMealLimit = errors.New("daily meal limit reached")
	ErrChatMealLimit  = errors.New("daily chat meal limit reached")
	ErrMealsInFlight  = errors.New("too many meals in flight")
)

type notFoundError struct{ what string }

func (e notFoundError) Error() string { return e.what + " not found" }
func (e notFoundError) Unwrap() error { return ErrNotFound }

func wrapNotFound(what string) error { return notFoundError{what: what} }
