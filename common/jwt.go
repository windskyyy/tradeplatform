package common

import (
	"TradePlatform/model"
	"github.com/dgrijalva/jwt-go"
	"github.com/liangyaopei/bloom"
	"time"
)

var jwtKey = []byte("a_secret_crect")

type Claims struct {
	UserId uint
	UserEmail string
	RecommendBloomFilter *bloom.Filter
	jwt.StandardClaims
}

func ReleaseToken(user model.User) (string, error)  {
	expirationTime := time.Now().Add(7 * 24 * time.Hour) // todo xutianmeng 时间太久了吧?
	filter := bloom.New(16384, 3, true)
	claims := &Claims{
		//UserId:         user.ID,
		RecommendBloomFilter: filter,
		UserEmail:      user.UserEmail,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt: time.Now().Unix(),
			Issuer: "oceanlearn.tech",
			Subject: "user token",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ParseToken(tokenString string) (*jwt.Token, *Claims, error){
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (i interface{}, err error) {
		return jwtKey, nil
	})

	return token, claims, err
}