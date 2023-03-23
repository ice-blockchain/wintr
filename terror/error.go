// SPDX-License-Identifier: ice License 1.0

package terror

import (
	"github.com/pkg/errors"
)

func New(err error, data map[string]any) *Err {
	return &Err{error: err, Data: data}
}

func As(err error) *Err {
	tErr := new(Err)
	if ok := errors.As(err, tErr); ok {
		return tErr
	}

	return nil
}

func (e *Err) Is(er error) bool {
	return errors.Is(er, e.error)
}

func (e *Err) Unwrap() error {
	return e.error
}

func (e *Err) As(err any) bool {
	o, ok := err.(*Err)
	if ok {
		*o = *e
	}

	return ok
}
