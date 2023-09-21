// GoToSocial
// Copyright (C) GoToSocial Authors admin@gotosocial.org
// SPDX-License-Identifier: AGPL-3.0-or-later
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package util

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
)

const (
	/* API version keys */

	APIVersionKey = "api_version"
	APIv1         = "v1"
	APIv2         = "v2"

	/* Common keys */

	IDKey      = "id"
	LimitKey   = "limit"
	LocalKey   = "local"
	MaxIDKey   = "max_id"
	SinceIDKey = "since_id"
	MinIDKey   = "min_id"

	/* Search keys */

	SearchExcludeUnreviewedKey = "exclude_unreviewed"
	SearchFollowingKey         = "following"
	SearchLookupKey            = "acct"
	SearchOffsetKey            = "offset"
	SearchQueryKey             = "q"
	SearchResolveKey           = "resolve"
	SearchTypeKey              = "type"

	/* Tag keys */

	TagNameKey = "tag_name"

	/* Web endpoint keys */

	WebUsernameKey = "username"
	WebStatusIDKey = "status"

	/* Domain permission keys */

	DomainPermissionExportKey = "export"
	DomainPermissionImportKey = "import"
)

// parseError returns gtserror.WithCode set to 400 Bad Request, to indicate
// to the caller that a key was set to a value that could not be parsed.
func parseError(key string, value, defaultValue any, err error) gtserror.WithCode {
	err = fmt.Errorf("error parsing key %s with value %s as %T: %w", key, value, defaultValue, err)
	return gtserror.NewErrorBadRequest(err, err.Error())
}

// requiredError returns gtserror.WithCode set to 400 Bad Request, to indicate
// to the caller a required key value was not provided, or was empty.
func requiredError(key string) gtserror.WithCode {
	err := fmt.Errorf("required key %s was not set or had empty value", key)
	return gtserror.NewErrorBadRequest(err, err.Error())
}

/*
	Parse functions for *OPTIONAL* parameters with default values.
*/

func ParseLimit(value string, defaultValue int, max, min int) (int, gtserror.WithCode) {
	i, err := parseInt(value, defaultValue, max, min, LimitKey)
	if err != nil {
		return 0, err
	}

	return i, nil
}

func ParseLocal(value string, defaultValue bool) (bool, gtserror.WithCode) {
	return parseBool(value, defaultValue, LocalKey)
}

func ParseMaxID(value string, defaultValue string) string {
	if value == "" {
		return defaultValue
	}

	return value
}

func ParseSearchExcludeUnreviewed(value string, defaultValue bool) (bool, gtserror.WithCode) {
	return parseBool(value, defaultValue, SearchExcludeUnreviewedKey)
}

func ParseSearchFollowing(value string, defaultValue bool) (bool, gtserror.WithCode) {
	return parseBool(value, defaultValue, SearchFollowingKey)
}

func ParseSearchOffset(value string, defaultValue int, max, min int) (int, gtserror.WithCode) {
	return parseInt(value, defaultValue, max, min, SearchOffsetKey)
}

func ParseSearchResolve(value string, defaultValue bool) (bool, gtserror.WithCode) {
	return parseBool(value, defaultValue, SearchResolveKey)
}

func ParseDomainPermissionExport(value string, defaultValue bool) (bool, gtserror.WithCode) {
	return parseBool(value, defaultValue, DomainPermissionExportKey)
}

func ParseDomainPermissionImport(value string, defaultValue bool) (bool, gtserror.WithCode) {
	return parseBool(value, defaultValue, DomainPermissionImportKey)
}

/*
	Parse functions for *REQUIRED* parameters.
*/

func ParseAPIVersion(value string, availableVersion ...string) (string, gtserror.WithCode) {
	key := APIVersionKey

	if value == "" {
		return "", requiredError(key)
	}

	for _, av := range availableVersion {
		if value == av {
			return value, nil
		}
	}

	err := fmt.Errorf(
		"invalid API version, valid versions for this path are [%s]",
		strings.Join(availableVersion, ", "),
	)
	return "", gtserror.NewErrorBadRequest(err, err.Error())
}

func ParseID(value string) (string, gtserror.WithCode) {
	key := IDKey

	if value == "" {
		return "", requiredError(key)
	}

	return value, nil
}

func ParseSearchLookup(value string) (string, gtserror.WithCode) {
	key := SearchLookupKey

	if value == "" {
		return "", requiredError(key)
	}

	return value, nil
}

func ParseSearchQuery(value string) (string, gtserror.WithCode) {
	key := SearchQueryKey

	if value == "" {
		return "", requiredError(key)
	}

	return value, nil
}

func ParseTagName(value string) (string, gtserror.WithCode) {
	key := TagNameKey

	if value == "" {
		return "", requiredError(key)
	}

	return value, nil
}

func ParseWebUsername(value string) (string, gtserror.WithCode) {
	key := WebUsernameKey

	if value == "" {
		return "", requiredError(key)
	}

	return value, nil
}

func ParseWebStatusID(value string) (string, gtserror.WithCode) {
	key := WebStatusIDKey

	if value == "" {
		return "", requiredError(key)
	}

	return value, nil
}

/*
	Internal functions
*/

func parseBool(value string, defaultValue bool, key string) (bool, gtserror.WithCode) {
	if value == "" {
		return defaultValue, nil
	}

	i, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue, parseError(key, value, defaultValue, err)
	}

	return i, nil
}

func parseInt(value string, defaultValue int, max int, min int, key string) (int, gtserror.WithCode) {
	if value == "" {
		return defaultValue, nil
	}

	i, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue, parseError(key, value, defaultValue, err)
	}

	if i > max {
		i = max
	} else if i < min {
		i = min
	}

	return i, nil
}
