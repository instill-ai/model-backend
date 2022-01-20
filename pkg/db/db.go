package db

import (
	"fmt"
	"time"

	"github.com/instill-ai/model-backend/configs"
	utils "github.com/instill-ai/model-backend/internal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() {
	databaseConfig := configs.Config.Database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=%s",
		databaseConfig.Host,
		databaseConfig.Username,
		databaseConfig.Password,
		databaseConfig.Name,
		databaseConfig.Port,
		databaseConfig.TimeZone,
	)
	var err error
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{
		QueryFields: true, // QueryFields mode will select by all fields’ name for current model
	})

	if err != nil {
		panic("Could not open database connection")
	}

	sqlDB, _ := db.DB()

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDB.SetMaxIdleConns(databaseConfig.Pool.IdleConnections)
	// SetMaxOpenConns sets the maximum number of open connections to the database.
	sqlDB.SetMaxOpenConns(databaseConfig.Pool.MaxConnections)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDB.SetConnMaxLifetime(time.Minute * databaseConfig.Pool.ConnLifeTime)

	DB = db
}

func Close() {

	// https://github.com/go-gorm/gorm/issues/3216
	//
	// This only works with a single master connection, but when dealing with replicas using DBResolver,
	// it does not close everything since DB.DB() only returns the master connection.
	if DB != nil {
		sqlDB, _ := DB.DB()

		sqlDB.Close()
	}
}

func GetNameDesFromId(id string) (string, string) {
	for i := 0; i < len(utils.ModelNames); i++ {
		model := utils.ModelNames[i]
		if model["id"] == id {
			return fmt.Sprintf("%v", model["name"]), fmt.Sprintf("%v", model["description"])
		}
	}
	return id, ""
}
