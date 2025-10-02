// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mattn/go-sqlite3"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SQLiteDbUserVersion This is the latest internal version of the SQLite Igor database. The value
// should map to the most recent db schema.
const SQLiteDbUserVersion = 2 // (for Igor 2.2)
//const SQLiteDbUserVersion = 1 // (for Igor 2.1)
//const SQLiteDbUserVersion = 0   // (for Igor 2.0)

func init() {
	sql.Register("sqlite3_igor",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				_, err := conn.Exec("PRAGMA case_sensitive_like = ON", nil)
				return err
			},
		})
}

// NewSqliteGormBackend returns the instantiation of the implementation
func NewSqliteGormBackend() IGormDb {

	sqliteDbLoc := filepath.Join(igor.Database.DbFolderPath, "igor.db")
	var isNewDB = false

	if _, err := os.Stat(sqliteDbLoc); errors.Is(err, os.ErrNotExist) {
		if file, crErr := os.OpenFile(sqliteDbLoc, os.O_CREATE, 0640); crErr != nil {
			exitPrintFatal(fmt.Sprintf("%v", crErr))
		} else {
			_ = file.Close()
			isNewDB = true
		}
	}

	logger.Info().Msgf("opening database session at %s", sqliteDbLoc)

	dial := &sqlite.Dialector{
		DriverName: "sqlite3_igor",
		DSN:        sqliteDbLoc,
	}

	db, err := gorm.Open(dial, stdGormConfig)
	if err != nil {
		exitPrintFatal(fmt.Sprintf("%v", err))
	}

	sqlDB, sqlDbErr := db.DB()
	if sqlDbErr != nil {
		exitPrintFatal(fmt.Sprintf("%v", err))
	}

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(20)
	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(100)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Hour)

	if isNewDB {
		_, err = sqlDB.Exec(fmt.Sprintf("PRAGMA user_version = %d", SQLiteDbUserVersion))
		if err != nil {
			exitPrintFatal(fmt.Sprintf("%v", err))
		}
		logger.Info().Msgf("created new sqlite database with current schema (version %d)", SQLiteDbUserVersion)

	} else {
		var userVersion int
		row, _ := sqlDB.Query("PRAGMA user_version")
		for row.Next() {
			_ = row.Scan(&userVersion)
		}
		_ = row.Close()
		if userVersion != SQLiteDbUserVersion {
			exitPrintFatal(fmt.Sprintf("%v", fmt.Errorf("database version = %d, should be %d :: run database upgrade script and re-start", userVersion, SQLiteDbUserVersion)))
		} else {
			logger.Info().Msgf("database schema version = %d", userVersion)
		}
	}

	logger.Debug().Msg("auto-migrating GORM models...")
	err = db.AutoMigrate(&Permission{}, &User{}, &Group{}, &Host{}, &HostPolicy{}, &Cluster{}, &Reservation{}, &Kickstart{}, &Distro{}, &Profile{}, &DistroImage{}, &HistoryRecord{}, &MaintenanceRes{})
	if err != nil {
		exitPrintFatal(fmt.Sprintf("%v", err))
	}
	logger.Debug().Msg("auto-migration finished")

	return &GormBackend{
		Database: db,
	}
}
