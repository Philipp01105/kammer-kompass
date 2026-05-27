package bootstrap

import (
	"context"

	"github.com/Philipp01105/kammer-kompass/backend/internal/db/sqlc"
	ihkcatalog "github.com/Philipp01105/kammer-kompass/backend/internal/ihk_catalog"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SyncIHKCatalog(ctx context.Context, db *pgxpool.Pool) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := sqlc.New(db).WithTx(tx)
	for _, entry := range ihkcatalog.All() {
		city := entry.City
		officialURL := entry.OfficialURL
		ihk, err := q.UpsertCatalogIHK(ctx, sqlc.UpsertCatalogIHKParams{
			Name:        entry.Name,
			Slug:        entry.Slug,
			City:        &city,
			State:       entry.State,
			OfficialUrl: &officialURL,
		})
		if err != nil {
			return err
		}
		if err := q.EnsureEmptyIHKInfoPage(ctx, ihk.ID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
