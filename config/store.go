package config

import (
	"github.com/clear-ness/qa-discussion/model"
)

type Store interface {
	Get() *model.Config
}
