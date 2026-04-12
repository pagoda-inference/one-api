package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type SignIn struct {
	Id        int   `json:"id"`
	UserId    int   `json:"user_id" gorm:"index"`
	Date      string `json:"date" gorm:"type:varchar(10);index"` // format: 2006-01-02
	Quota     int64 `json:"quota" gorm:"bigint"`                 // reward quota
	CreatedAt int64 `json:"created_at" gorm:"bigint"`
}

// SignInRecord represents a sign-in record for API response
type SignInRecord struct {
	Date      string `json:"date"`
	Status    string `json:"status"` // "completed" or "none"
	Quota     int64  `json:"quota"`
}

func GetSignInRecords(userId int, startDate string, endDate string) ([]*SignInRecord, error) {
	var signIns []*SignIn
	query := DB.Model(&SignIn{}).Where("user_id = ?", userId)

	if startDate != "" {
		query = query.Where("date >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("date <= ?", endDate)
	}

	err := query.Order("date asc").Find(&signIns).Error
	if err != nil {
		return nil, err
	}

	records := make([]*SignInRecord, len(signIns))
	for i, s := range signIns {
		records[i] = &SignInRecord{
			Date:      s.Date,
			Status:    "completed",
			Quota:     s.Quota,
		}
	}
	return records, nil
}

func GetSignInByDate(userId int, date string) (*SignIn, error) {
	var signIn SignIn
	err := DB.Where("user_id = ? AND date = ?", userId, date).First(&signIn).Error
	if err != nil {
		return nil, err
	}
	return &signIn, nil
}

func CreateSignIn(signIn *SignIn) error {
	// Check if already signed in today
	exists, err := SignInExists(signIn.UserId, signIn.Date)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("今日已签到")
	}
	return DB.Create(signIn).Error
}

func SignInExists(userId int, date string) (bool, error) {
	var count int64
	err := DB.Model(&SignIn{}).Where("user_id = ? AND date = ?", userId, date).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func GetUserQuotaForSignIn(userId int) (int64, error) {
	var user User
	err := DB.First(&user, userId).Error
	if err != nil {
		return 0, err
	}
	return user.Quota, nil
}

func AddUserQuota(userId int, quota int64) error {
	return DB.Model(&User{}).Where("id = ?", userId).Update("quota", gorm.Expr("quota + ?", quota)).Error
}

// GetTodaySignInReward returns the sign-in reward quota
// In production, this could come from config or be based on user level
func GetTodaySignInReward() int64 {
	return 100 * 1000 // 100K quota as sign-in reward
}

// TodayDate returns today's date in "2006-01-02" format
func TodayDate() string {
	return time.Now().Format("2006-01-02")
}