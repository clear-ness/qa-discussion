package configservice

import (
	"github.com/clear-ness/qa-discussion/model"
)

type ConfigService interface {
	Config() *model.Config
}
