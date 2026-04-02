package notification

import "gorm.io/gorm"

// GormUserRepo retrieves user OpenID from the users table via GORM.
type GormUserRepo struct {
	db *gorm.DB
}

// NewGormUserRepo creates a GormUserRepo with the given database connection.
func NewGormUserRepo(db *gorm.DB) *GormUserRepo {
	return &GormUserRepo{db: db}
}

// GetUserOpenID returns the WeChat OpenID for the given user ID.
// Returns empty string (no error) if the user exists but has no OpenID.
func (r *GormUserRepo) GetUserOpenID(userID uint) (string, error) {
	var openID *string
	err := r.db.Table("users").Select("openid").Where("id = ?", userID).Scan(&openID).Error
	if err != nil {
		return "", err
	}
	if openID == nil {
		return "", nil
	}
	return *openID, nil
}
