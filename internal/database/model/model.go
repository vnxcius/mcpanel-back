package model

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	*gorm.Model
	Password string `gorm:"type:varchar(255);notnull"`
}

type Session struct {
	ID           string `gorm:"type:varchar(255);notnull;primaryKey"`
	UserID       uint   `gorm:"notnull;"`
	RefreshToken string `gorm:"type:varchar(255);notnull"`
	IsRevoked    bool   `gorm:"default:false;notnull"`
	CreatedAt    time.Time
	ExpiresAt    time.Time

	User User
}
