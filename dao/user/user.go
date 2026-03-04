package user

import (
	"github.com/nanjiek/GopherAI_HybridAgent/common/mysql"
	"github.com/nanjiek/GopherAI_HybridAgent/model"
	"context"

	"gorm.io/gorm"
)

const (
	CodeMsg     = "GopherAI验证码如下(验证码仅限于2分钟有效): "
	UserNameMsg = "GopherAI的账号如下，请保留好，后续可以用账号进行登录 "
)

var ctx = context.Background()

// 这边只能通过账号进行登录
func IsExistUser(username string) (bool, *model.User) {

	user, err := mysql.GetUserByUsername(username)

	if err == gorm.ErrRecordNotFound || user == nil {
		return false, nil
	}

	return true, user
}

func Register(username, email, password string) (*model.User, bool) {
	if user, err := mysql.InsertUser(&model.User{
		Email:    email,
		Name:     username,
		Username: username,
		Password: password,
	}); err != nil {
		return nil, false
	} else {
		return user, true
	}
}

func UpdateUserPassword(userID int64, passwordHash string) bool {
	return mysql.DB.Model(&model.User{}).Where("id = ?", userID).Update("password", passwordHash).Error == nil
}
