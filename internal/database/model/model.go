package model

import "gorm.io/gorm"

type Example struct {
	*gorm.Model

	Name string `gorm:"type:varchar(50);notnull"`
	Role string `gorm:"type:varchar(10);notnull;default:user"`
}
