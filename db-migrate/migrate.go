package main

import (
	"crypto/sha256"
	"database/sql"
	"flag"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	LatestDbVersion     = 1 // CHANGE THIS when a new upgrade is needed
	IgorConfHome        = "/etc/igor/"
	IgorConfFileDefault = "igor-server.yaml"
	IgorConfPathDefault = IgorConfHome + IgorConfFileDefault
)

var (
	igorHome       string
	configFilepath = flag.String("config", "", "path to server configuration file")
	config         Config
	err            error
	igorDB         *sql.DB
	userVersion    int
	backupPath     string
)

func main() {

	if igorHome = os.Getenv("IGOR_HOME"); strings.TrimSpace(igorHome) == "" {
		fmt.Fprintf(os.Stderr, "error: environment variable IGOR_HOME not defined\n")
		os.Exit(1)
	}

	flag.Parse()
	initConfig(configFilepath)
	if config.Database.Adapter != "sqlite" {
		fmt.Fprintf(os.Stderr, "database.adapter setting '%s' is not allowed; must be 'sqlite'", config.Database.Adapter)
		os.Exit(1)
	}
	if config.Database.DbFolderPath == "" {
		config.Database.DbFolderPath = filepath.Join(igorHome, ".database")
		fmt.Printf("database.dbFolderPath not specified, using default (IGOR_HOME): %s\n", config.Database.DbFolderPath)
	}
	sqliteDbLoc := filepath.Join(config.Database.DbFolderPath, "igor.db")

	_, pathErr := os.Stat(sqliteDbLoc)
	if pathErr != nil {
		fmt.Fprintf(os.Stderr, "%v\n", pathErr)
		os.Exit(1)
	}

	sql.Register("sqlite3_upgrade_db",
		&sqlite3.SQLiteDriver{
			ConnectHook: func(conn *sqlite3.SQLiteConn) error {
				_, err = conn.Exec("PRAGMA case_sensitive_like = ON", nil)
				return err
			},
		})

	igorDB, _ = sql.Open("sqlite3_upgrade_db", sqliteDbLoc)
	defer igorDB.Close()

	row, _ := igorDB.Query("PRAGMA user_version")
	for row.Next() {
		_ = row.Scan(&userVersion)
		fmt.Printf("Current Igor DB version = %d\n", userVersion)
	}
	_ = row.Close()

	if userVersion == LatestDbVersion {
		fmt.Printf("No upgrade performed on Igor DB. Version %d is current.\n", userVersion)
		return
	} else {
		fmt.Printf("Will attempt incremental upgrade of Igor DB to latest version (%d)\n", LatestDbVersion)
		if err = makeBackup(sqliteDbLoc, userVersion); err != nil {
			fmt.Fprintf(os.Stderr, "aborting ... couldn't create backup of sqlite db: %v\n", err)
			os.Exit(1)
		}
	}

	var isUpgraded = false
	var sqlFile string

	if userVersion == 0 {
		sqlFile = getNextSqlFile("migrations/migrate0to1.sql")
		fmt.Println("Executing upgrade step: 0 -> 1")
		if isUpgraded, err = runNextUpgradeStep(sqlFile); err != nil {
			fmt.Fprintf(os.Stderr, "There was an issue performing the upgrade: %v\n", err)
			restoreBackup(sqliteDbLoc, backupPath)
			os.Exit(1)
		}
	}

	// For each subsequent file, do another step
	if userVersion == 1 {
		// repeat formula from above
	}

	if isUpgraded {
		_, err = igorDB.Exec(fmt.Sprintf("PRAGMA user_version = %d", userVersion))
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Upgrade complete to DB version %d\n", userVersion)
	} else {
		fmt.Printf("No upgrade performed on Igor DB. Version %d is current.\n", userVersion)
	}

}

type Config struct {
	Database struct {
		Adapter      string `yaml:"adapter" json:"adapter"`
		DbFolderPath string `yaml:"dbFolderPath" json:"dbFolderPath"` // only used for SQLite
	} `yaml:"database" json:"database"`
}

func initConfig(configPath *string) {

	configLocHome := filepath.Join(igorHome, "conf", IgorConfFileDefault)

	if *configPath != "" {
		fmt.Printf("Looking for conf file at location %s", *configPath)
		if _, errArg := os.Stat(*configPath); errArg == nil {
			fmt.Printf(" ... found\n")
			config = readConfig(*configPath)
			return
		}
	}

	fmt.Printf("\nLooking for conf file at location %s", IgorConfPathDefault)
	if _, errEtc := os.Stat(IgorConfPathDefault); errEtc == nil {
		fmt.Printf(" ... found\n")
		config = readConfig(IgorConfPathDefault)
		return
	}

	fmt.Printf("\nLooking for conf file at location %s", configLocHome)
	if _, errHome := os.Stat(configLocHome); errHome == nil {
		fmt.Printf(" ... found\n")
		config = readConfig(configLocHome)
		return
	}
}

func makeBackup(src string, oldVersion int) (err error) {

	var srcFile *os.File
	var dbBytes []byte

	dbBytes, err = os.ReadFile(src)
	if err != nil {
		return err
	}

	h := sha256.New()
	_, _ = io.Copy(h, srcFile)
	backupHash := h.Sum(nil)
	backupPath = fmt.Sprintf(src+".%d.%x.backup", oldVersion, backupHash[:4])

	err = os.WriteFile(backupPath, dbBytes, 0664)
	if err != nil {
		return err
	}

	return nil
}

func restoreBackup(src, backupPath string) {
	_ = os.Rename(backupPath, src)
}

func readConfig(path string) (c Config) {

	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to open config file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err = yaml.NewDecoder(f).Decode(&c); err != nil {
		fmt.Fprintf(os.Stderr, "unable to parse config file: %v\n", err)
		os.Exit(1)
	}
	return
}

func getNextSqlFile(filepath string) string {
	sqlFileBytes, err := os.ReadFile(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		fmt.Fprintf(os.Stderr, "trying the Igor scripts directory\n")
		if sqlFileBytes, err = os.ReadFile("scripts/" + filepath); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	}
	return string(sqlFileBytes)
}

func runNextUpgradeStep(sqlFile string) (success bool, txErr error) {
	success = false
	tx, _ := igorDB.Begin()
	if _, txErr = igorDB.Exec(sqlFile); txErr != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			fmt.Fprintf(os.Stderr, "rollback error! : %v\n", rollbackErr)
			txErr = fmt.Errorf("upgrade step failed and rollback failed! : %v", txErr)
			// if we make a copy of the igor.db file and stash it away, we can replace corrupted upgrade
		} else {
			txErr = fmt.Errorf("upgrade step failed but rollback succeeded: %v", txErr)
		}
	} else {
		if txErr = tx.Commit(); txErr != nil {
			return
		}
		success = true
		userVersion++
	}
	return
}
