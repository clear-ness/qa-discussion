package storetest

import (
	"flag"
	"fmt"
	"os"

	"github.com/clear-ness/qa-discussion/model"
	"github.com/go-sql-driver/mysql"
)

const (
	defaultMysqlDSN     = "root@tcp(localhost:3306)/qa_discussion_test?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s"
	defaultMysqlRootPWD = ""
)

func getEnv(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	} else {
		return defaultValue
	}
}

func log(message string) {
	verbose := false
	if verboseFlag := flag.Lookup("test.v"); verboseFlag != nil {
		verbose = verboseFlag.Value.String() != ""
	}
	if verboseFlag := flag.Lookup("v"); verboseFlag != nil {
		verbose = verboseFlag.Value.String() != ""
	}

	if verbose {
		fmt.Println(message)
	}
}

func databaseSettings(driver, dataSource string) *model.SqlSettings {
	settings := &model.SqlSettings{
		DriverName:                  &driver,
		DataSource:                  &dataSource,
		DataSourceReplicas:          []string{},
		MaxIdleConns:                new(int),
		ConnMaxLifetimeMilliseconds: new(int),
		MaxOpenConns:                new(int),
		Trace:                       model.NewBool(false),
		QueryTimeout:                new(int),
	}
	*settings.MaxIdleConns = 10
	*settings.ConnMaxLifetimeMilliseconds = 3600000
	*settings.MaxOpenConns = 100
	*settings.QueryTimeout = 60

	return settings
}

func MySQLSettings() *model.SqlSettings {
	dsn := getEnv("TEST_DATABASE_MYSQL_DSN", defaultMysqlDSN)
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		panic("failed to parse dsn " + dsn + ": " + err.Error())
	}

	return databaseSettings("mysql", cfg.FormatDSN())
}

func MakeSqlSettings() *model.SqlSettings {
	var settings *model.SqlSettings

	settings = MySQLSettings()

	return settings
}
