package storage

import (
	"github.com/palarix/exponential/internal/model"
)

func ReadEvents() ([]model.Event, error) {
	return ReadEventsRef()
}

func ValidateEvents() (int, error) {
	return ValidateEventsRef()
}

func ReadArchivedEvents() ([]model.Event, error) {
	return ReadArchivedEventsRef()
}
