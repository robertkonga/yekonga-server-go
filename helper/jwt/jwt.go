package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Token struct {
	Header    map[string]interface{}
	Payload   map[string]interface{}
	Signature string
}

const secretKey = "Yekonga-256-bit-secret"

func EncodeJWT(claims map[string]interface{}) (string, error) {
	// Header
	header := map[string]interface{}{
		"alg": "HS256",
		"typ": "JWT",
	}
	headerJSON, _ := json.Marshal(header)
	headerBase64 := base64.URLEncoding.EncodeToString(headerJSON)

	// Payload
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	payloadJSON, _ := json.Marshal(claims)
	payloadBase64 := base64.URLEncoding.EncodeToString(payloadJSON)

	// Signature
	signatureInput := headerBase64 + "." + payloadBase64
	signature := generateSignature(signatureInput)

	return signatureInput + "." + signature, nil
}

func DecodeJWT(tokenString string) (bool, map[string]interface{}) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return false, nil
	}

	headerBase64 := parts[0]
	payloadBase64 := parts[1]
	signatureProvided := parts[2]

	// Verify signature
	signatureInput := headerBase64 + "." + payloadBase64
	expectedSignature := generateSignature(signatureInput)
	if signatureProvided != expectedSignature {
		return false, nil
	}

	// Decode payload
	payloadJSON, err := base64.URLEncoding.DecodeString(payloadBase64)
	if err != nil {
		return false, nil
	}

	var payload map[string]interface{}
	json.Unmarshal(payloadJSON, &payload)

	// Check expiration
	if exp, ok := payload["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return false, nil
		}
	}

	return true, payload
}

func generateSignature(input string) string {
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(input))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

// Example usage
func ExampleUsage() {
	claims := map[string]interface{}{
		"sub":  "1234567890",
		"name": "John Doe",
	}

	// Create token
	tokenString, err := EncodeJWT(claims)
	if err != nil {
		fmt.Println("Error creating token:", err)
		return
	}

	// Validate token
	valid, payload := DecodeJWT(tokenString)
	if valid {
		fmt.Println("Token is valid:", payload)
	} else {
		fmt.Println("Token is invalid")
	}
}
