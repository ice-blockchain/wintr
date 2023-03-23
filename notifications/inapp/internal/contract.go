// SPDX-License-Identifier: ice License 1.0

package internal

import (
	"github.com/ice-blockchain/wintr/time"
)

// Public API.

type (
	Parcel struct {
		Time        *time.Time     `json:"time,omitempty" example:"2022-01-03T16:20:52.156534Z"`
		ReferenceID string         `json:"referenceId,omitempty" example:"e5335afb-8ec4-4669-953d-37f0c712ba8d"`
		Data        map[string]any `json:"data,omitempty"`
		Action      string         `json:"action,omitempty" example:"broadcast_news"`
		Actor       ID             `json:"actor,omitempty"`
		Subject     ID             `json:"subject,omitempty"`
	}
	ID struct {
		Type  string `json:"type" example:"userId"`
		Value string `json:"value" example:"e5335afb-8ec4-4669-953d-37f0c712ba8d"`
	}
)
