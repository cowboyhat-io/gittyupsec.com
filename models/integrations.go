package models

import "github.com/jinzhu/gorm"

const (
	ErrOrgRequired    modelError = "models: org is required"
	ErrUserIDRequired modelError = "models: id is required"
)

// Integration represents the integration a user can upload
type Integration struct {
	gorm.Model
	UserID uint   `gorm:"not_null;index"`
	Org    string `gorm:"not_null"`
	Token  string `gorm:"not_null"`
}

type integrationValFn func(*Integration) error

type integrationService struct {
	IntegrationDB
}

type integrationValidator struct {
	IntegrationDB
}

// IntegrationService is for interacting with the
// Integration table
type IntegrationService interface {
	IntegrationDB
}

// IntegrationDB is for CRUD operations involving the table
type IntegrationDB interface {
	ByID(id uint) (*Integration, error)
	ByUserID(userID uint) ([]Integration, error)
	Create(integration *Integration) error
	Update(integration *Integration) error
	Delete(id uint) error
}

// ByID finds a integration by its unique ID
func (gg *integrationGorm) ByID(id uint) (*Integration, error) {
	var integration Integration
	db := gg.db.Where("id = ?", id)
	err := first(db, &integration)
	if err != nil {
		return nil, err
	}
	return &integration, nil
}

// ByUserID returns the user's integrations on the index page
func (gg *integrationGorm) ByUserID(userID uint) ([]Integration, error) {
	var integrations []Integration
	// We build this query *exactly* the same way we build
	// a query for a single user.
	db := gg.db.Where("user_id = ?", userID)
	// The real difference is in using Find instead of First
	// and passing in a slice instead of a single integration as
	// the argument.
	if err := db.Find(&integrations).Error; err != nil {
		return nil, err
	}
	return integrations, nil
}

type integrationGorm struct {
	db *gorm.DB
}

func (gg *integrationGorm) Create(integration *Integration) error {
	return gg.db.Create(integration).Error
}

func (gv *integrationValidator) Create(integration *Integration) error {
	err := runIntegrationValFns(integration,
		gv.userIDRequired,
		gv.orgRequired)
	if err != nil {
		return err
	}
	return gv.IntegrationDB.Create(integration)
}

func (gg *integrationGorm) Update(integration *Integration) error {
	return gg.db.Save(integration).Error
}

func (gv *integrationValidator) Update(integration *Integration) error {
	err := runIntegrationValFns(integration,
		gv.userIDRequired,
		gv.orgRequired)
	if err != nil {
		return err
	}
	return gv.IntegrationDB.Update(integration)
}

func (gg *integrationGorm) Delete(id uint) error {
	integration := Integration{Model: gorm.Model{ID: id}}
	return gg.db.Delete(&integration).Error
}

func (gv *integrationValidator) nonZeroID(integration *Integration) error {
	if integration.ID <= 0 {
		return ErrIDInvalid
	}
	return nil
}

func (gv *integrationValidator) Delete(id uint) error {
	var integration Integration
	integration.ID = id
	if err := runIntegrationValFns(&integration, gv.nonZeroID); err != nil {
		return err
	}
	return gv.IntegrationDB.Delete(integration.ID)
}

var _ IntegrationDB = &integrationGorm{}

// NewIntegrationService returns a IntegrationService
// for use with the Integration table object(s)
func NewIntegrationService(db *gorm.DB) IntegrationService {
	return &integrationService{
		IntegrationDB: &integrationValidator{
			IntegrationDB: &integrationGorm{
				db: db,
			},
		},
	}
}

func runIntegrationValFns(integration *Integration, fns ...integrationValFn) error {
	for _, fn := range fns {
		if err := fn(integration); err != nil {
			return err
		}
	}
	return nil
}

func (gv *integrationValidator) userIDRequired(g *Integration) error {
	if g.UserID <= 0 {
		return ErrUserIDRequired
	}
	return nil
}

func (gv *integrationValidator) orgRequired(g *Integration) error {
	if g.Org == "" {
		return ErrOrgRequired
	}
	return nil
}
