package mysql

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DSN            string
	MaxOpenConns   int
	MaxIdleConns   int
	MaxLifetimeSec time.Duration
	LogMode        logger.LogLevel
}

type Client struct {
	db *gorm.DB
}

func NewClient(cfg *Config) (*Client, error) {
	logLevel := logger.Silent
	if cfg.LogMode > 0 {
		logLevel = cfg.LogMode
	}

	db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MySQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}

	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}

	if cfg.MaxLifetimeSec > 0 {
		sqlDB.SetConnMaxLifetime(cfg.MaxLifetimeSec)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping MySQL: %w", err)
	}

	return &Client{db: db}, nil
}

func (c *Client) GetDB() *gorm.DB {
	return c.db
}

func (c *Client) Close() error {
	sqlDB, err := c.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
