package storage

import (
	"github.com/palarix/exponential/internal/model"
)

func AppendEvent(event model.Event) error {
	return AppendEventRef(event)
}
