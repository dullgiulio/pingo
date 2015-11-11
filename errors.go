// Copyright 2015 Giulio Iotti. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package pingo

import (
	"errors"
	"strings"
)

const (
	errorCodeConnFailed = "err-connection-failed"
	errorCodeHttpServe  = "err-http-serve"
)

// Error reported when connection to the external plugin has failed.
type ErrConnectionFailed error

// Error reported when the external plugin cannot start listening for calls.
type ErrHttpServe error

// Error reported when an invalid message is printed by the external plugin.
type ErrInvalidMessage error

// Error reported when the plugin fails to register before the registration
// timeout expires.
type ErrRegistrationTimeout error

func parseError(line string) error {
	parts := strings.SplitN(line, ": ", 2)
	if parts[0] == "" {
		return nil
	}

	err := errors.New(parts[1])

	switch parts[0] {
	case errorCodeConnFailed:
		return ErrConnectionFailed(err)
	case errorCodeHttpServe:
		return ErrHttpServe(err)
	}

	return err
}
