package main

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt"
)

// TODO Add tests to cover the http requests
// TODO Add more tests to cover the error cases

func TestDecodeSecretToPem(t *testing.T) {
    samplePemEncodedText, _ := ioutil.ReadFile("testdata/sample_key_pem_encoded.txt")
    samplePemText, _ := ioutil.ReadFile("testdata/sample_key.pem")

    pem, err := DecodeSecretToPem(string(samplePemEncodedText))
    if err != nil {
        t.Errorf("err actual %q, expected %q", err, "nil")
    }

    s1 := strings.Replace(pem, "\n", "", -1)
    s2 := strings.Replace(string(samplePemText), "\n", "", -1)
    if s1 != s2 {
        t.Error("Difference pem text retrieved")
        fmt.Println("Actual:")
        fmt.Println(s1)
        fmt.Println("Expected:")
        fmt.Println(s2)
    }
}

func TestDecodeSecretToPemErr(t *testing.T) {
    actualValue, err := DecodeSecretToPem("dummy-secret-id")
    if actualValue != "" || !strings.Contains(err.Error(), "illegal base64 data at input byte 5") {
        t.Errorf("err actual %q, expected %q", err, "illegal base64 data at input byte 5")
    }
}

func TestCreateJwtToken(t *testing.T) {
    samplePemText, _ := ioutil.ReadFile("testdata/sample_key.pem")
    privateKey, _ := jwt.ParseRSAPrivateKeyFromPEM(samplePemText)

    appId := "dummy-issuer"
    tokenString, _ := CreateJwtToken(&appId, privateKey)
    fmt.Println(tokenString)

    token2, _ := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
        return privateKey, nil
    })
    claims, ok := token2.Claims.(*jwt.StandardClaims)

    if ok != true {
        t.Error("not ok")
    }

    if claims.Issuer != appId {
        t.Errorf("claims.Issuer actual %q, expected %q", claims.Issuer, appId)
    }

    if claims.ExpiresAt-claims.IssuedAt != 600 {
        t.Errorf("claims.ExpiresAt-claims.IssuedAt actual %q, expected %q", claims.ExpiresAt-claims.IssuedAt, "600")
    }
}
