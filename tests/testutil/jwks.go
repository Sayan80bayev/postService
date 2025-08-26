package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	privateKey *rsa.PrivateKey
	publicKey  rsa.PublicKey
)

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func StartMockJWKS(addr string) {
	var err error
	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	publicKey = privateKey.PublicKey

	jwks := JWKS{
		Keys: []JWK{
			{
				Kty: "RSA",
				Alg: "RS256",
				Use: "sig",
				Kid: "test-key-id",
				N:   base64Url(publicKey.N.Bytes()),
				E:   base64Url(big.NewInt(int64(publicKey.E)).Bytes()),
			},
		},
	}

	http.HandleFunc("/certs", func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(jwks)
		if err != nil {
			return
		}
	})

	go func() {
		if err := http.ListenAndServe(addr, nil); err != nil {
			panic(err)
		}
	}()
}

func base64Url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// GenerateMockToken creates a JWT signed with the mock private key
func GenerateMockToken(userId string) string {
	now := time.Now()
	claims := jwt.MapClaims{
		"exp": now.Add(time.Hour).Unix(),
		"iat": now.Unix(),
		"jti": "onrtro:cba1c0d6-c951-a2c4-53be-2c964da5125b",
		"iss": "http://go-project-keycloak:8080/realms/go-project",
		"aud": []string{"realm-management", "account"},
		"sub": userId,
		"typ": "Bearer",
		"azp": "auth_service",
		"sid": "f2ec6ae1-ef38-4b4f-9554-ef992d255d59",
		"acr": "1",
		"allowed-origins": []string{
			"http://localhost:8082",
			"/*",
		},
		"realm_access": map[string]interface{}{
			"roles": []string{
				"default-roles-go-project",
				"offline_access",
				"uma_authorization",
			},
		},
		"resource_access": map[string]interface{}{
			"realm-management": map[string]interface{}{
				"roles": []string{"manage-users"},
			},
			"auth_service": map[string]interface{}{
				"roles": []string{"user"},
			},
			"account": map[string]interface{}{
				"roles": []string{"manage-account", "manage-account-links", "view-profile"},
			},
		},
		"scope":              "profile email",
		"email_verified":     false,
		"preferred_username": "sayan123serv@gmail.com",
		"email":              "sayan123serv@gmail.com",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key-id"
	ss, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
	return ss
}
