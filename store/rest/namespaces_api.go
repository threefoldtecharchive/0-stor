package rest

import (
	log "github.com/Sirupsen/logrus"
	"github.com/zero-os/0-stor/store/config"
	"github.com/zero-os/0-stor/store/db"
	"github.com/zero-os/0-stor/store/rest/models"
)

var _ (NamespacesInterface) = (*NamespacesAPI)(nil)

// NamespacesAPI is API implementation of /namespaces root endpoint
type NamespacesAPI struct {
	config        config.Settings
	db            db.DB
	jwtKey        []byte
	storeStatsMgr *models.StoreStatMgr
}

func NewNamespacesAPI(db db.DB, conf config.Settings) *NamespacesAPI {
	return &NamespacesAPI{
		db:            db,
		config:        conf,
		storeStatsMgr: models.NewStoreStatMgr(db),
	}

}

func (api NamespacesAPI) DB() db.DB {
	return api.db
}

func (api NamespacesAPI) Config() config.Settings {
	return api.config
}

func (api NamespacesAPI) JWTKey() []byte {
	return api.jwtKey
}

func (api NamespacesAPI) namespaceStat(label string) (*models.NamespaceStats, error) {
	namespaceStats := &models.NamespaceStats{
		Namespace: label,
	}

	b, err := api.db.Get(namespaceStats.Key())
	if err != nil {
		log.Errorln(err.Error())
		return nil, err
	}

	if err = namespaceStats.Decode(b); err != nil {
		log.Errorln(err.Error())
		return nil, err
	}

	return namespaceStats, nil
}

func (api NamespacesAPI) setNamespaceStat(namespaceStats *models.NamespaceStats) error {
	b, err := namespaceStats.Encode()
	if err != nil {
		return err
	}

	return api.db.Set(namespaceStats.Key(), b)
}

func (api NamespacesAPI) reservation(id string, namespace string) (*models.Reservation, error) {
	reservation := &models.Reservation{
		Namespace: namespace,
		Id:        id,
	}

	b, err := api.db.Get(reservation.Key())
	if err != nil {
		return nil, err
	}

	if err := reservation.Decode(b); err != nil {
		return nil, err
	}

	return reservation, nil
}

func (api NamespacesAPI) setReservation(reservation *models.Reservation) error {
	b, err := reservation.Encode()
	if err != nil {
		return err
	}
	return api.db.Set(reservation.Key(), b)
}
