package dao

import (
	"OMEGA3-IOT/internal/model"
)

type InstanceDao interface {
	Save(instance *model.Instance) error
	Get(string) (*model.Instance, error)
}
