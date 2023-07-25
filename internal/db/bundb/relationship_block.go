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
	"fmt"

	"github.com/superseriousbusiness/gotosocial/internal/db"
	"github.com/superseriousbusiness/gotosocial/internal/gtscontext"
	"github.com/superseriousbusiness/gotosocial/internal/gtsmodel"
	"github.com/uptrace/bun"
)

func (r *relationshipDB) IsBlocked(ctx context.Context, sourceAccountID string, targetAccountID string) (bool, error) {
	block, err := r.GetBlock(
		gtscontext.SetBarebones(ctx),
		sourceAccountID,
		targetAccountID,
	)
	if err != nil && !errors.Is(err, db.ErrNoEntries) {
		return false, err
	}
	return (block != nil), nil
}

func (r *relationshipDB) IsEitherBlocked(ctx context.Context, accountID1 string, accountID2 string) (bool, error) {
	// Look for a block in direction of account1->account2
	b1, err := r.IsBlocked(ctx, accountID1, accountID2)
	if err != nil || b1 {
		return true, err
	}

	// Look for a block in direction of account2->account1
	b2, err := r.IsBlocked(ctx, accountID2, accountID1)
	if err != nil || b2 {
		return true, err
	}

	return false, nil
}

func (r *relationshipDB) GetBlockByID(ctx context.Context, id string) (*gtsmodel.Block, error) {
	return r.getBlock(
		ctx,
		"ID",
		func(block *gtsmodel.Block) error {
			return r.db.NewSelect().Model(block).
				Where("? = ?", bun.Ident("block.id"), id).
				Scan(ctx)
		},
		id,
	)
}

func (r *relationshipDB) GetBlockByURI(ctx context.Context, uri string) (*gtsmodel.Block, error) {
	return r.getBlock(
		ctx,
		"URI",
		func(block *gtsmodel.Block) error {
			return r.db.NewSelect().Model(block).
				Where("? = ?", bun.Ident("block.uri"), uri).
				Scan(ctx)
		},
		uri,
	)
}

func (r *relationshipDB) GetBlock(ctx context.Context, sourceAccountID string, targetAccountID string) (*gtsmodel.Block, error) {
	return r.getBlock(
		ctx,
		"AccountID.TargetAccountID",
		func(block *gtsmodel.Block) error {
			return r.db.NewSelect().Model(block).
				Where("? = ?", bun.Ident("block.account_id"), sourceAccountID).
				Where("? = ?", bun.Ident("block.target_account_id"), targetAccountID).
				Scan(ctx)
		},
		sourceAccountID,
		targetAccountID,
	)
}

func (r *relationshipDB) getBlock(ctx context.Context, lookup string, dbQuery func(*gtsmodel.Block) error, keyParts ...any) (*gtsmodel.Block, error) {
	// Fetch block from cache with loader callback
	block, err := r.state.Caches.GTS.Block().Load(lookup, func() (*gtsmodel.Block, error) {
		var block gtsmodel.Block

		// Not cached! Perform database query
		if err := dbQuery(&block); err != nil {
			return nil, r.db.ProcessError(err)
		}

		return &block, nil
	}, keyParts...)
	if err != nil {
		// already processe
		return nil, err
	}

	if gtscontext.Barebones(ctx) {
		// Only a barebones model was requested.
		return block, nil
	}

	// Set the block source account
	block.Account, err = r.state.DB.GetAccountByID(
		gtscontext.SetBarebones(ctx),
		block.AccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting block source account: %w", err)
	}

	// Set the block target account
	block.TargetAccount, err = r.state.DB.GetAccountByID(
		gtscontext.SetBarebones(ctx),
		block.TargetAccountID,
	)
	if err != nil {
		return nil, fmt.Errorf("error getting block target account: %w", err)
	}

	return block, nil
}

func (r *relationshipDB) PutBlock(ctx context.Context, block *gtsmodel.Block) error {
	return r.state.Caches.GTS.Block().Store(block, func() error {
		_, err := r.db.NewInsert().Model(block).Exec(ctx)
		return r.db.ProcessError(err)
	})
}

func (r *relationshipDB) DeleteBlockByID(ctx context.Context, id string) error {
	defer r.state.Caches.GTS.Block().Invalidate("ID", id)

	// Load block into cache before attempting a delete,
	// as we need it cached in order to trigger the invalidate
	// callback. This in turn invalidates others.
	_, err := r.GetBlockByID(gtscontext.SetBarebones(ctx), id)
	if err != nil {
		if errors.Is(err, db.ErrNoEntries) {
			// not an issue.
			err = nil
		}
		return err
	}

	// Finally delete block from DB.
	_, err = r.db.NewDelete().
		Table("blocks").
		Where("? = ?", bun.Ident("id"), id).
		Exec(ctx)
	return r.db.ProcessError(err)
}

func (r *relationshipDB) DeleteBlockByURI(ctx context.Context, uri string) error {
	defer r.state.Caches.GTS.Block().Invalidate("URI", uri)

	// Load block into cache before attempting a delete,
	// as we need it cached in order to trigger the invalidate
	// callback. This in turn invalidates others.
	_, err := r.GetBlockByURI(gtscontext.SetBarebones(ctx), uri)
	if err != nil {
		if errors.Is(err, db.ErrNoEntries) {
			// not an issue.
			err = nil
		}
		return err
	}

	// Finally delete block from DB.
	_, err = r.db.NewDelete().
		Table("blocks").
		Where("? = ?", bun.Ident("uri"), uri).
		Exec(ctx)
	return r.db.ProcessError(err)
}

func (r *relationshipDB) DeleteAccountBlocks(ctx context.Context, accountID string) error {
	var blockIDs []string

	// Get full list of IDs.
	if err := r.db.NewSelect().
		Column("id").
		Table("blocks").
		WhereOr("? = ? OR ? = ?",
			bun.Ident("account_id"),
			accountID,
			bun.Ident("target_account_id"),
			accountID,
		).
		Scan(ctx, &blockIDs); err != nil {
		return r.db.ProcessError(err)
	}

	defer func() {
		// Invalidate all IDs on return.
		for _, id := range blockIDs {
			r.state.Caches.GTS.Block().Invalidate("ID", id)
		}
	}()

	// Load all blocks into cache, this *really* isn't great
	// but it is the only way we can ensure we invalidate all
	// related caches correctly (e.g. visibility).
	for _, id := range blockIDs {
		_, err := r.GetBlockByID(ctx, id)
		if err != nil && !errors.Is(err, db.ErrNoEntries) {
			return err
		}
	}

	// Finally delete all from DB.
	_, err := r.db.NewDelete().
		Table("blocks").
		Where("? IN (?)", bun.Ident("id"), bun.In(blockIDs)).
		Exec(ctx)
	return r.db.ProcessError(err)
}
