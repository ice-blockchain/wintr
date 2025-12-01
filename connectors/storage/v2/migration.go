// SPDX-License-Identifier: ice License 1.0

package storage

import (
	"context"
	"fmt"
	"io/fs"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tern/v2/migrate"

	"github.com/ice-blockchain/wintr/log"
)

func NewStringDDL(ddl string) DDL {
	return &stringDDL{Data: ddl}
}

func NewFilesystemDDL(fs fs.FS, schemaTableName string) DDL {
	return &filesystemDDL{FS: fs, SchemeTable: schemaTableName}
}

func (d *stringDDL) run(ctx context.Context, pool *pgxpool.Pool) error {
	for statement := range strings.SplitSeq(d.Data, "----") {
		_, err := pool.Exec(ctx, statement)
		if !ignorableDDLError(err) {
			return fmt.Errorf("failed to execute DDL statement: %v: %w", statement, maskError(err))
		}
	}
	return nil
}

func (d *filesystemDDL) run(ctx context.Context, pool *pgxpool.Pool) error {
	schemaTable := d.SchemeTable

	if schemaTable == "" {
		const defaultSchemaTable = "wintr_storagev2_schema_migrations"
		log.Info("schema table name not provided for migrations, using default: " + defaultSchemaTable)
		schemaTable = defaultSchemaTable
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("cannot acquire connection for migration: %w", err)
	}
	defer conn.Release()

	m, err := migrate.NewMigrator(ctx, conn.Conn(), schemaTable)
	if err != nil {
		return fmt.Errorf("cannot create migrator: %w", err)
	}

	err = m.LoadMigrations(d.FS)
	if err != nil {
		return fmt.Errorf("cannot load migrations from fs: %w", err)
	}

	if v, err := m.GetCurrentVersion(ctx); err == nil {
		log.Info(fmt.Sprintf("current schema version: %d", v))
	}

	if err := doAfterConnect(ctx, "0", conn.Conn()); err != nil {
		return fmt.Errorf("cannot set session parameters before migration: %w", err)
	}

	defer func() {
		if derr := doAfterConnect(ctx, "", conn.Conn()); derr != nil {
			log.Error(fmt.Errorf("cannot reset session parameters after migration: %w", derr))
		}
	}()

	m.OnStart = func(sequence int32, name, direction, sql string) {
		log.Info(fmt.Sprintf("starting migration: %d: %s: %s", sequence, name, direction))
	}

	return m.Migrate(ctx)
}
