package main

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/golang-jwt/jwt"
)

var Version = "v1.0.0"

func CheckError(message string, err error, statusCode int) {
    if err != nil {
        log.Println(err)
        fmt.Println("ERROR:", message)
        os.Exit(statusCode)
    }
}

func GetSecret(iamRoleArn *string, awsRegion *string, secretId *string) (string, error) {
    // Initial credentials loaded from SDK's default credential chain. Such as
    // the environment, shared credentials (~/.aws/credentials), or EC2 Instance
    // Role. These credentials will be used to to make the STS Assume Role API.
    cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(*awsRegion))
    if err != nil {
        fmt.Println("AWS Configuration Error", err)
		return "", err
    }

    // Create the credentials from AssumeRoleProvider to assume the role.
    stsSvc := sts.NewFromConfig(cfg)
    provider := stscreds.NewAssumeRoleProvider(stsSvc, *iamRoleArn)
    cfg.Credentials = aws.NewCredentialsCache(provider)

    // Create service client value configured for credentials from assumed role.
    svc := secretsmanager.NewFromConfig(cfg)
    result, err := svc.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{
        SecretId: aws.String(*secretId),
    })
    if err != nil {
        return "", err
    }

    var dat map[string]interface{}
    if err := json.Unmarshal([]byte(*result.SecretString), &dat); err != nil {
        return "", err
    }
    return DecodeSecretToPem(dat["pem"].(string))
}

func DecodeSecretToPem(inputData string) (string, error) {
    pem, err := base64.StdEncoding.DecodeString(inputData)
    if err != nil {
        return "", err
    }
    return string(pem), nil
}

func CreateJwtToken(appId *string, privateKey *rsa.PrivateKey) (string, error) {
    // Get the JWT token for the given GitHub APP ID and private key
    claims := &jwt.StandardClaims{
        ExpiresAt: time.Now().Add(time.Second * 600).Unix(),
        IssuedAt:  time.Now().Unix(),
        Issuer:    *appId,
    }

    token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

    jwtToken, err := token.SignedString(privateKey)
    if err != nil {
        return "", err
    }
    return jwtToken, err
}

// The actual response contains more attributes but we only need 'token'
type ApiResponse struct {
    Token string
}

func ProcessApiRequest(req *http.Request) (string, error) {
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    var respObj ApiResponse
    err2 := json.NewDecoder(resp.Body).Decode(&respObj)
    if err2 != nil {
        return "", err2
    }
    return respObj.Token, err
}

func CreateAppAccessToken(jwtToken *string, installId *string) (string, error) {
    // Get App token, which is set on the GitHub app
    ghApiUrl := "https://api.github.com/app/installations/" + *installId + "/access_tokens"
    req, err := http.NewRequest("POST", ghApiUrl, nil)
    if err != nil {
        return "", err
    }
    req.Header.Set("Authorization", "Bearer " + *jwtToken)
    req.Header.Set("Accept", "application/vnd.github.v3+json")

    token, err := ProcessApiRequest(req)
    return token, err
}

func CreateRunnerRegoToken(appToken *string, orgName *string) (string, error) {
    // Get the runner registration token, a short lived token for (only) adding the runners
    ghApiUrl := "https://api.github.com/orgs/" + *orgName + "/actions/runners/registration-token"
    req, err := http.NewRequest("POST", ghApiUrl, nil)
    if err != nil {
        return "", err
    }
    req.Header.Set("Authorization", "token " + *appToken)
    req.Header.Set("Accept", "application/vnd.github.machine-man-preview+json")

    token, err := ProcessApiRequest(req)
    return token, err
}

func main() {
    // Parse command line flags
    awsRegion := flag.String("awsRegion", "ap-southeast-2", "AWS Region (same as that of IAM Role)")
    iamRoleArn := flag.String("iamRoleArn", "", "ARN of IAM Role with secret read permission")
    appId := flag.String("appId", "", "GitHub App ID")
    installId := flag.String("installId", "", "GitHub Install ID")
    orgName := flag.String("orgName", "", "GitHub Org Name")
    secretArn := flag.String("secretArn", "", "ARN of GitHub Runner Secret")
    showVersion := flag.Bool("version", false, "Show version of this app")
    flag.Parse()
    if *showVersion {
        fmt.Println(Version)
        os.Exit(0)
    }
    if *appId == "" {
        CheckError("Missing -appId", fmt.Errorf("Missing -appId"), 2)
    }
    if *installId == "" {
        CheckError("Missing -installId", fmt.Errorf("Missing -installId"), 2)
    }
    if *orgName == "" {
        CheckError("Missing -orgName", fmt.Errorf("Missing -orgName"), 2)
    }
    if *iamRoleArn == "" {
        CheckError("Missing -iamRoleArn", fmt.Errorf("Missing -iamRoleArn"), 2)
    }
    if *secretArn == "" {
        CheckError("Missing -secretArn", fmt.Errorf("Missing -secretArn"), 2)
    }

    // Retrieve PEM storing at Secret Manager
    certPemStr, err := GetSecret(iamRoleArn, awsRegion, secretArn)
    CheckError("Cannot get secret value", err, 2)

    // Get private key from PEM
    privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(certPemStr))
    CheckError("Cannot get private key", err, 2)

    // Get the JWT token for the given GitHub APP ID and private key
    jwtToken, err := CreateJwtToken(appId, privateKey)
    CheckError("Cannot get jwt token", err, 2)

    // Get App token, which is set on the GitHub app
    appToken, err := CreateAppAccessToken(&jwtToken, installId)
    CheckError("Cannot get app token", err, 2)

    // Get the runner registration token, a short lived token for (only) adding the runners
    regoToken, err := CreateRunnerRegoToken(&appToken, orgName)
    CheckError("Cannot get runner rego token", err, 2)

    fmt.Println(regoToken)
}
