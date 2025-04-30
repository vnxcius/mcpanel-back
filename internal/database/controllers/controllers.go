package controllers

import (
	"github.com/vnxcius/sss-backend/internal/database/model"
	"github.com/vnxcius/sss-backend/internal/database/pg"
)

func GetUser() (*model.User, error) {
	db := pg.DB
	var user model.User
	err := db.First(&user, 1).Error
	return &user, err
}

func CreateSession(s *model.Session) error {
	db := pg.DB
	return db.Create(s).Error
}

func DeleteSession(id string) error {
	db := pg.DB
	return db.Delete(&model.Session{}, "id = ?", id).Error
}

func GetSession(id string) (*model.Session, error) {
	db := pg.DB
	var session model.Session
	err := db.First(&session, "id = ?", id).Error
	return &session, err
}

func RevokeSession(id string) error {
	db := pg.DB
	// Check if any field was affected, if not return error
	return db.Model(&model.Session{}).Where("id = ?", id).Update("is_revoked", true).Error
}
