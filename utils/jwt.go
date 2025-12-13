package utils

import (
	"errors"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

const defaultSignKey = "123456"
const defaultIssuer = "go_verdor"
const defaultExpire = time.Hour * 24
const defaultAutoRenew = true // 自动续期

var JWT = &jwtInfo{
	signKey:   defaultSignKey,
	issuer:    defaultIssuer,
	expire:    defaultExpire,
	autoRenew: defaultAutoRenew,
}

type jwtInfo struct {
	signKey   string        // 签名密钥
	issuer    string        // 签发者
	expire    time.Duration // 过期时间
	autoRenew bool          // 自动续期，超过一半时间自动续期
}

func (j *jwtInfo) GetSignKey() string {
	return j.signKey
}

func (j *jwtInfo) SetSignKey(key string) {
	j.signKey = key
}

func (j *jwtInfo) GetIssuer() string {
	return j.issuer
}

func (j *jwtInfo) SetIssuer(issuer string) {
	j.issuer = issuer
}

func (j *jwtInfo) GetExpire() time.Duration {
	return j.expire
}

func (j *jwtInfo) SetExpire(expire time.Duration) {
	j.expire = expire
}

func (j *jwtInfo) GetAutoRenew() bool {
	return j.autoRenew
}

func (j *jwtInfo) SetAutoRenew(autoRenew bool) {
	j.autoRenew = autoRenew
}

// GenerateToken 生成JWT token
func (j *jwtInfo) GenerateToken(Info string) (string, error) {
	claims := CustomClaims{
		Info: Info,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(j.expire).Unix(),
			Issuer:    j.issuer,
			IssuedAt:  time.Now().Unix(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.signKey))
}

// ParseToken 解析JWT token
func (j *jwtInfo) ParseToken(tokenString string) (*CustomClaims, string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.signKey), nil
	})
	if err != nil {
		return nil, "", err
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		if j.autoRenew && claims.ExpiresAt-time.Now().Unix() < int64(j.expire)/2 {
			// 自动续期，超过一半时间自动续期
			newToken, err := j.GenerateToken(claims.Info)
			if err != nil {
				return nil, "", err
			}
			return claims, newToken, nil
		}
		return claims, "", nil
	}

	return nil, "", errors.New("invalid token")
}

type CustomClaims struct {
	Info string
	jwt.StandardClaims
}

// Tmp 测试JWT
func Tmp() {
	// 生成token
	token, err := JWT.GenerateToken("123")
	if err != nil {
		panic(err)
	}
	println(token)

	// 解析token
	claims, newToken, err := JWT.ParseToken(token)
	if err != nil {
		panic(err)
	}
	println(claims.Info)
	if newToken != "" {
		println("newToken:", newToken)
	}
}
