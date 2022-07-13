// SPDX-License-Identifier: BUSL-1.1

package storage

import (
	"time"

	"github.com/pkg/errors"
)

const IndexName = "indexName"

var (
	ErrNotFound                 = errors.New("not found")
	ErrRelationNotFound         = errors.New("relation not found")
	ErrDuplicate                = errors.New("duplicate")
	ErrRetryOnInvalidForeignKey = errors.New("unexpected error when inserting or updating a entry with an invalid foreign key reference")
	ErrDDLInvalid               = errors.New("DDL is invalid")
	ErrNoSpacesConfigured       = errors.New("no spaces configured")
)

// Private API.

const (
	dbSchemaInitDeadline         = 30 * time.Second
	getAllUserSpacesFunctionName = "get_all_user_spaces"
)

//nolint:gochecknoglobals // Because its loaded once, at runtime.
var cfg config

type (
	config struct {
		DB struct {
			User     string   `yaml:"user"`
			Password string   `yaml:"password"`
			URLs     []string `yaml:"urls"`
			Spaces   []string `yaml:"spaces"`
			ReadOnly bool     `yaml:"readOnly"`
		} `yaml:"db"`
	}
)
