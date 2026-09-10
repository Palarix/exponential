package storage

import (
	"github.com/palarix/exponential/internal/model"
)

func AppendEventCollapsed(event model.Event) error {
	return AppendEventRef(event)
}
