package handler

import "gorm.io/gorm"

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	DB *gorm.DB
}
