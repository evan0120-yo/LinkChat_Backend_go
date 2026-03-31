package validator

import (
	"errors"
)

type LinkValidator interface {
	ValidateCreateLink(requesterID, targetID string) error
}

type linkValidator struct{}

func NewLinkValidator() LinkValidator {
	return &linkValidator{}
}

func (v *linkValidator) ValidateCreateLink(requesterID, targetID string) error {
	if requesterID == "" || targetID == "" {
		return errors.New("requester_id and target_id cannot be empty")
	}
	if requesterID == targetID {
		return errors.New("cannot link with yourself")
	}
	return nil
}
