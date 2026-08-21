package db

import (
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open connects to the database using the driver named by cfgDriver and
// returns a ready-to-use *gorm.DB with the schema migrated.
//
// Only "sqlite" is wired up today (via the pure-Go glebarez/sqlite driver,
// which needs no C compiler). Switching to Postgres/MySQL later means adding
// a case here for gorm.io/driver/postgres or gorm.io/driver/mysql — models
// and query code elsewhere do not change.
func Open(cfgDriver, dsn string) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch cfgDriver {
	case "sqlite", "":
		dialector = sqlite.Open(dsn)
	default:
		return nil, fmt.Errorf("unsupported DB_DRIVER %q (only \"sqlite\" is wired up so far)", cfgDriver)
	}

	gdb, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := gdb.AutoMigrate(
		&Track{},
		&Line{},
		&TranslationCache{},
		&ScrapeSource{},
	); err != nil {
		return nil, fmt.Errorf("automigrate: %w", err)
	}

	return gdb, nil
}
