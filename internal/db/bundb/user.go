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

package bundb

import (
	"context"
	"errors"
	"time"

	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtscontext"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/state"
	"github.com/uptrace/bun"
)

type userDB struct {
	db    *WrappedDB
	state *state.State
}

func (u *userDB) GetUserByID(ctx context.Context, id string) (*gtsmodel.User, error) {
	return u.state.Caches.GTS.User().Load("ID", func() (*gtsmodel.User, error) {
		var user gtsmodel.User

		q := u.db.
			NewSelect().
			Model(&user).
			Relation("Account").
			Where("? = ?", bun.Ident("user.id"), id)

		if err := q.Scan(ctx); err != nil {
			return nil, u.db.ProcessError(err)
		}

		return &user, nil
	}, id)
}

func (u *userDB) GetUserByAccountID(ctx context.Context, accountID string) (*gtsmodel.User, error) {
	return u.state.Caches.GTS.User().Load("AccountID", func() (*gtsmodel.User, error) {
		var user gtsmodel.User

		q := u.db.
			NewSelect().
			Model(&user).
			Relation("Account").
			Where("? = ?", bun.Ident("user.account_id"), accountID)

		if err := q.Scan(ctx); err != nil {
			return nil, u.db.ProcessError(err)
		}

		return &user, nil
	}, accountID)
}

func (u *userDB) GetUserByEmailAddress(ctx context.Context, emailAddress string) (*gtsmodel.User, error) {
	return u.state.Caches.GTS.User().Load("Email", func() (*gtsmodel.User, error) {
		var user gtsmodel.User

		q := u.db.
			NewSelect().
			Model(&user).
			Relation("Account").
			Where("? = ?", bun.Ident("user.email"), emailAddress)

		if err := q.Scan(ctx); err != nil {
			return nil, u.db.ProcessError(err)
		}

		return &user, nil
	}, emailAddress)
}

func (u *userDB) GetUserByExternalID(ctx context.Context, id string) (*gtsmodel.User, error) {
	return u.state.Caches.GTS.User().Load("ExternalID", func() (*gtsmodel.User, error) {
		var user gtsmodel.User

		q := u.db.
			NewSelect().
			Model(&user).
			Relation("Account").
			Where("? = ?", bun.Ident("user.external_id"), id)

		if err := q.Scan(ctx); err != nil {
			return nil, u.db.ProcessError(err)
		}

		return &user, nil
	}, id)
}

func (u *userDB) GetUserByConfirmationToken(ctx context.Context, confirmationToken string) (*gtsmodel.User, error) {
	return u.state.Caches.GTS.User().Load("ConfirmationToken", func() (*gtsmodel.User, error) {
		var user gtsmodel.User

		q := u.db.
			NewSelect().
			Model(&user).
			Relation("Account").
			Where("? = ?", bun.Ident("user.confirmation_token"), confirmationToken)

		if err := q.Scan(ctx); err != nil {
			return nil, u.db.ProcessError(err)
		}

		return &user, nil
	}, confirmationToken)
}

func (u *userDB) GetAllUsers(ctx context.Context) ([]*gtsmodel.User, error) {
	var users []*gtsmodel.User
	q := u.db.
		NewSelect().
		Model(&users).
		Relation("Account")

	if err := q.Scan(ctx); err != nil {
		return nil, u.db.ProcessError(err)
	}

	return users, nil
}

func (u *userDB) PutUser(ctx context.Context, user *gtsmodel.User) error {
	return u.state.Caches.GTS.User().Store(user, func() error {
		_, err := u.db.
			NewInsert().
			Model(user).
			Exec(ctx)
		return u.db.ProcessError(err)
	})
}

func (u *userDB) UpdateUser(ctx context.Context, user *gtsmodel.User, columns ...string) error {
	// Update the user's last-updated
	user.UpdatedAt = time.Now()

	if len(columns) > 0 {
		// If we're updating by column, ensure "updated_at" is included
		columns = append(columns, "updated_at")
	}

	return u.state.Caches.GTS.User().Store(user, func() error {
		_, err := u.db.
			NewUpdate().
			Model(user).
			Where("? = ?", bun.Ident("user.id"), user.ID).
			Column(columns...).
			Exec(ctx)
		return u.db.ProcessError(err)
	})
}

func (u *userDB) DeleteUserByID(ctx context.Context, userID string) error {
	defer u.state.Caches.GTS.User().Invalidate("ID", userID)

	// Load user into cache before attempting a delete,
	// as we need it cached in order to trigger the invalidate
	// callback. This in turn invalidates others.
	_, err := u.GetUserByID(gtscontext.SetBarebones(ctx), userID)
	if err != nil {
		if errors.Is(err, db.ErrNoEntries) {
			// not an issue.
			err = nil
		}
		return err
	}

	// Finally delete user from DB.
	_, err = u.db.NewDelete().
		TableExpr("? AS ?", bun.Ident("users"), bun.Ident("user")).
		Where("? = ?", bun.Ident("user.id"), userID).
		Exec(ctx)
	return u.db.ProcessError(err)
}
