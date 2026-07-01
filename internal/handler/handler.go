package handler

import (
	"github.com/dreynaldis/pokechamps-logger/internal/config"
	"gorm.io/gorm"
)

// Handler holds shared dependencies for all HTTP handlers.
type Handler struct {
	DB     *gorm.DB
	Config *config.Config
}
