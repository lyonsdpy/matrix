package loader

import (
	"matrix/api/domain"
	"matrix/api/internal/repository"

	"github.com/vikstrous/dataloadgen"
)

type ctxKey struct{}

type DeviceAppsKey struct {
	DeviceID string
	Limit    int
}

type DeviceIPsKey struct {
	DeviceID string
	Limit    int
}

type AppPortsKey struct {
	AppID string
	Limit int
}

type DeviceLinksKey struct {
	DeviceID string
	Limit    int
}

type Loaders struct {
	IpsByDeviceID      *dataloadgen.Loader[DeviceAppsKey, []*domain.IPv4Addr]
	DevLinksByDeviceID *dataloadgen.Loader[DeviceLinksKey, []*domain.DeviceLink]
}

func NewLoaders(repo *repository.Neo4jRepo) *Loaders {
	return &Loaders{
		// TODO
	}
}
