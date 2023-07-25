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
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/superseriousbusiness/gotosocial/internal/ap"
	"github.com/superseriousbusiness/gotosocial/internal/config"
	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtserror"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/superseriousbusiness/gotosocial/internal/id"
	"github.com/superseriousbusiness/gotosocial/internal/log"
	"github.com/superseriousbusiness/gotosocial/internal/state"
	"github.com/superseriousbusiness/gotosocial/internal/uris"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

// generate RSA keys of this length
const rsaKeyBits = 2048

type adminDB struct {
	db    *WrappedDB
	state *state.State
}

func (a *adminDB) IsUsernameAvailable(ctx context.Context, username string) (bool, error) {
	q := a.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("accounts"), bun.Ident("account")).
		Column("account.id").
		Where("? = ?", bun.Ident("account.username"), username).
		Where("? IS NULL", bun.Ident("account.domain"))
	return a.db.NotExists(ctx, q)
}

func (a *adminDB) IsEmailAvailable(ctx context.Context, email string) (bool, error) {
	// parse the domain from the email
	m, err := mail.ParseAddress(email)
	if err != nil {
		return false, fmt.Errorf("error parsing email address %s: %s", email, err)
	}
	domain := strings.Split(m.Address, "@")[1] // domain will always be the second part after @

	// check if the email domain is blocked
	emailDomainBlockedQ := a.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("email_domain_blocks"), bun.Ident("email_domain_block")).
		Column("email_domain_block.id").
		Where("? = ?", bun.Ident("email_domain_block.domain"), domain)
	emailDomainBlocked, err := a.db.Exists(ctx, emailDomainBlockedQ)
	if err != nil {
		return false, err
	}
	if emailDomainBlocked {
		return false, fmt.Errorf("email domain %s is blocked", domain)
	}

	// check if this email is associated with a user already
	q := a.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("users"), bun.Ident("user")).
		Column("user.id").
		Where("? = ?", bun.Ident("user.email"), email).
		WhereOr("? = ?", bun.Ident("user.unconfirmed_email"), email)
	return a.db.NotExists(ctx, q)
}

func (a *adminDB) NewSignup(ctx context.Context, newSignup gtsmodel.NewSignup) (*gtsmodel.User, error) {
	// If something went wrong previously while doing a new
	// sign up with this username, we might already have an
	// account, so check first.
	account, err := a.state.DB.GetAccountByUsernameDomain(ctx, newSignup.Username, "")
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		// Real error occurred.
		err := gtserror.Newf("error checking for existing account: %w", err)
		return nil, err
	}

	// If we didn't yet have an account
	// with this username, create one now.
	if account == nil {
		uris := uris.GenerateURIsForAccount(newSignup.Username)

		accountID, err := id.NewRandomULID()
		if err != nil {
			err := gtserror.Newf("error creating new account id: %w", err)
			return nil, err
		}

		privKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
		if err != nil {
			err := gtserror.Newf("error creating new rsa private key: %w", err)
			return nil, err
		}

		account = &gtsmodel.Account{
			ID:                    accountID,
			Username:              newSignup.Username,
			DisplayName:           newSignup.Username,
			Reason:                newSignup.Reason,
			Privacy:               gtsmodel.VisibilityDefault,
			URI:                   uris.UserURI,
			URL:                   uris.UserURL,
			InboxURI:              uris.InboxURI,
			OutboxURI:             uris.OutboxURI,
			FollowingURI:          uris.FollowingURI,
			FollowersURI:          uris.FollowersURI,
			FeaturedCollectionURI: uris.FeaturedCollectionURI,
			ActorType:             ap.ActorPerson,
			PrivateKey:            privKey,
			PublicKey:             &privKey.PublicKey,
			PublicKeyURI:          uris.PublicKeyURI,
		}

		// Insert the new account!
		if err := a.state.DB.PutAccount(ctx, account); err != nil {
			return nil, err
		}
	}

	// Created or already had an account.
	// Ensure user not already created.
	user, err := a.state.DB.GetUserByAccountID(ctx, account.ID)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		// Real error occurred.
		err := gtserror.Newf("error checking for existing user: %w", err)
		return nil, err
	}

	defer func() {
		// Pin account to (new)
		// user before returning.
		user.Account = account
	}()

	if user != nil {
		// Already had a user for this
		// account, just return that.
		return user, nil
	}

	// Had no user for this account, time to create one!
	newUserID, err := id.NewRandomULID()
	if err != nil {
		err := gtserror.Newf("error creating new user id: %w", err)
		return nil, err
	}

	encryptedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(newSignup.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		err := gtserror.Newf("error hashing password: %w", err)
		return nil, err
	}

	user = &gtsmodel.User{
		ID:                     newUserID,
		AccountID:              account.ID,
		Account:                account,
		EncryptedPassword:      string(encryptedPassword),
		SignUpIP:               newSignup.SignUpIP.To4(),
		Locale:                 newSignup.Locale,
		UnconfirmedEmail:       newSignup.Email,
		CreatedByApplicationID: newSignup.AppID,
		ExternalID:             newSignup.ExternalID,
	}

	if newSignup.EmailVerified {
		// Mark given email as confirmed.
		user.ConfirmedAt = time.Now()
		user.Email = newSignup.Email
	}

	trueBool := func() *bool { t := true; return &t }

	if newSignup.Admin {
		// Make new user mod + admin.
		user.Moderator = trueBool()
		user.Admin = trueBool()
	}

	if newSignup.PreApproved {
		// Mark new user as approved.
		user.Approved = trueBool()
	}

	// Insert the user!
	if err := a.state.DB.PutUser(ctx, user); err != nil {
		err := gtserror.Newf("db error inserting user: %w", err)
		return nil, err
	}

	return user, nil
}

func (a *adminDB) CreateInstanceAccount(ctx context.Context) error {
	username := config.GetHost()

	q := a.db.
		NewSelect().
		TableExpr("? AS ?", bun.Ident("accounts"), bun.Ident("account")).
		Column("account.id").
		Where("? = ?", bun.Ident("account.username"), username).
		Where("? IS NULL", bun.Ident("account.domain"))

	exists, err := a.db.Exists(ctx, q)
	if err != nil {
		return err
	}
	if exists {
		log.Infof(ctx, "instance account %s already exists", username)
		return nil
	}

	key, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		log.Errorf(ctx, "error creating new rsa key: %s", err)
		return err
	}

	aID, err := id.NewRandomULID()
	if err != nil {
		return err
	}

	newAccountURIs := uris.GenerateURIsForAccount(username)
	acct := &gtsmodel.Account{
		ID:                    aID,
		Username:              username,
		DisplayName:           username,
		URL:                   newAccountURIs.UserURL,
		PrivateKey:            key,
		PublicKey:             &key.PublicKey,
		PublicKeyURI:          newAccountURIs.PublicKeyURI,
		ActorType:             ap.ActorPerson,
		URI:                   newAccountURIs.UserURI,
		InboxURI:              newAccountURIs.InboxURI,
		OutboxURI:             newAccountURIs.OutboxURI,
		FollowersURI:          newAccountURIs.FollowersURI,
		FollowingURI:          newAccountURIs.FollowingURI,
		FeaturedCollectionURI: newAccountURIs.FeaturedCollectionURI,
	}

	// insert the new account!
	if err := a.state.DB.PutAccount(ctx, acct); err != nil {
		return err
	}

	log.Infof(ctx, "instance account %s CREATED with id %s", username, acct.ID)
	return nil
}

func (a *adminDB) CreateInstanceInstance(ctx context.Context) error {
	protocol := config.GetProtocol()
	host := config.GetHost()

	// check if instance entry already exists
	q := a.db.
		NewSelect().
		Column("instance.id").
		TableExpr("? AS ?", bun.Ident("instances"), bun.Ident("instance")).
		Where("? = ?", bun.Ident("instance.domain"), host)

	exists, err := a.db.Exists(ctx, q)
	if err != nil {
		return err
	}
	if exists {
		log.Infof(ctx, "instance entry already exists")
		return nil
	}

	iID, err := id.NewRandomULID()
	if err != nil {
		return err
	}

	i := &gtsmodel.Instance{
		ID:     iID,
		Domain: host,
		Title:  host,
		URI:    fmt.Sprintf("%s://%s", protocol, host),
	}

	insertQ := a.db.
		NewInsert().
		Model(i)

	_, err = insertQ.Exec(ctx)
	if err != nil {
		return a.db.ProcessError(err)
	}

	log.Infof(ctx, "created instance instance %s with id %s", host, i.ID)
	return nil
}
