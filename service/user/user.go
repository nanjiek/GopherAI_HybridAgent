package user

import (
	"github.com/nanjiek/GopherAI_HybridAgent/common/code"
	myemail "github.com/nanjiek/GopherAI_HybridAgent/common/email"
	myredis "github.com/nanjiek/GopherAI_HybridAgent/common/redis"
	"github.com/nanjiek/GopherAI_HybridAgent/dao/user"
	"github.com/nanjiek/GopherAI_HybridAgent/model"
	"github.com/nanjiek/GopherAI_HybridAgent/utils"
	"github.com/nanjiek/GopherAI_HybridAgent/utils/myjwt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func Login(username, password string) (string, code.Code) {
	var userInformation *model.User
	var ok bool
	if ok, userInformation = user.IsExistUser(username); !ok {
		return "", code.CodeUserNotExist
	}

	matched, legacyMD5 := verifyPassword(password, userInformation.Password)
	if !matched {
		return "", code.CodeInvalidPassword
	}

	// On successful legacy login, upgrade hash to bcrypt.
	if legacyMD5 {
		if newHash, err := hashPassword(password); err == nil {
			_ = user.UpdateUserPassword(userInformation.ID, newHash)
		}
	}

	token, err := myjwt.GenerateToken(userInformation.ID, userInformation.Username)
	if err != nil {
		return "", code.CodeServerBusy
	}
	return token, code.CodeSuccess
}

func Register(email, password, captcha string) (string, code.Code) {
	var ok bool
	var userInformation *model.User

	if ok, _ = user.IsExistUser(email); ok {
		return "", code.CodeUserExist
	}

	if ok, _ = myredis.CheckCaptchaForEmail(email, captcha); !ok {
		return "", code.CodeInvalidCaptcha
	}

	username := utils.GetRandomNumbers(11)
	passwordHash, err := hashPassword(password)
	if err != nil {
		return "", code.CodeServerBusy
	}

	if userInformation, ok = user.Register(username, email, passwordHash); !ok {
		return "", code.CodeServerBusy
	}

	if err = myemail.SendCaptcha(email, username, user.UserNameMsg); err != nil {
		return "", code.CodeServerBusy
	}

	token, err := myjwt.GenerateToken(userInformation.ID, userInformation.Username)
	if err != nil {
		return "", code.CodeServerBusy
	}

	return token, code.CodeSuccess
}

func SendCaptcha(email_ string) code.Code {
	sendCode := utils.GetRandomNumbers(6)
	if err := myredis.SetCaptchaForEmail(email_, sendCode); err != nil {
		return code.CodeServerBusy
	}

	if err := myemail.SendCaptcha(email_, sendCode, myemail.CodeMsg); err != nil {
		return code.CodeServerBusy
	}

	return code.CodeSuccess
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func verifyPassword(password string, storedHash string) (matched bool, legacyMD5 bool) {
	if strings.HasPrefix(storedHash, "$2a$") || strings.HasPrefix(storedHash, "$2b$") || strings.HasPrefix(storedHash, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password)) == nil, false
	}
	return storedHash == utils.MD5(password), true
}
