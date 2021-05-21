/*
   GoToSocial
   Copyright (C) 2021 GoToSocial Authors admin@gotosocial.org

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package user

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/superseriousbusiness/gotosocial/internal/api"
	"github.com/superseriousbusiness/gotosocial/internal/config"
	"github.com/superseriousbusiness/gotosocial/internal/message"
	"github.com/superseriousbusiness/gotosocial/internal/router"
	"github.com/superseriousbusiness/gotosocial/internal/util"
)

const (
	// UsernameKey is for account usernames.
	UsernameKey = "username"
	// StatusIDKey is for status IDs
	StatusIDKey = "status"
	// UsersBasePath is the base path for serving information about Users eg https://example.org/users
	UsersBasePath = "/" + util.UsersPath
	// UsersBasePathWithUsername is just the users base path with the Username key in it.
	// Use this anywhere you need to know the username of the user being queried.
	// Eg https://example.org/users/:username
	UsersBasePathWithUsername = UsersBasePath + "/:" + UsernameKey
	// UsersInboxPath is for serving POST requests to a user's inbox with the given username key.
	UsersInboxPath = UsersBasePathWithUsername + "/" + util.InboxPath
	// UsersFollowersPath is for serving GET request's to a user's followers list, with the given username key.
	UsersFollowersPath = UsersBasePathWithUsername + "/" + util.FollowersPath
	// UsersStatusPath is for serving GET requests to a particular status by a user, with the given username key and status ID
	UsersStatusPath = UsersBasePathWithUsername + "/" + util.StatusesPath + "/:" + StatusIDKey
)

// ActivityPubAcceptHeaders represents the Accept headers mentioned here:
// https://www.w3.org/TR/activitypub/#retrieving-objects
var ActivityPubAcceptHeaders = []string{
	`application/activity+json`,
	`application/ld+json; profile="https://www.w3.org/ns/activitystreams"`,
}

// Module implements the FederationAPIModule interface
type Module struct {
	config    *config.Config
	processor message.Processor
	log       *logrus.Logger
}

// New returns a new auth module
func New(config *config.Config, processor message.Processor, log *logrus.Logger) api.FederationModule {
	return &Module{
		config:    config,
		processor: processor,
		log:       log,
	}
}

// Route satisfies the RESTAPIModule interface
func (m *Module) Route(s router.Router) error {
	s.AttachHandler(http.MethodGet, UsersBasePathWithUsername, m.UsersGETHandler)
	s.AttachHandler(http.MethodPost, UsersInboxPath, m.InboxPOSTHandler)
	s.AttachHandler(http.MethodGet, UsersFollowersPath, m.FollowersGETHandler)
	s.AttachHandler(http.MethodGet, UsersStatusPath, m.StatusGETHandler)
	return nil
}
