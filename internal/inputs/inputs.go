// Package inputs re-exports shared input types from jsonio for backward
// compatibility. New code should import jsonio directly. This package will
// be removed once all CLI commands are migrated.
package inputs

import (
	"github.com/palarix/exponential/internal/jsonio"
	"github.com/palarix/exponential/internal/model"
)

type AddInput = jsonio.AddInput
type UpdateInput = jsonio.UpdateInput
type CommentInput = jsonio.CommentInput
type LinkInput = jsonio.LinkInput

func DecodeStrict(content string, dst interface{}) error {
	return jsonio.DecodeStrict(content, dst)
}

func ValidateStatus(s string) error {
	return jsonio.ValidateStatus(s)
}

func LinksToDependencies(links []LinkInput) ([]model.Dependency, error) {
	return jsonio.LinksToDependencies(links)
}

func UpdatePayloadEmpty(p model.UpdatePayload) bool {
	return jsonio.UpdatePayloadEmpty(p)
}
