# github-runner-utils

[![githubactions](https://github.com/kyhau/github-runner-utils/workflows/github-runner-utils-build/badge.svg)](https://github.com/kyhau/github-runner-utils/actions)

`github-runner-utils` returns the GitHub runner registration token for registering a self-hosted runner to GitHub.

## Usage

```
Usage of ./github-runner-utils:
  -appId string
        GitHub App ID
  -awsRegion string
        AWS Region (same as that of IAM Role) (default "ap-southeast-2")
  -iamRoleArn string
        ARN of IAM Role with secret read permission
  -installId string
        GitHub Install ID
  -orgName string
        GitHub Org Name
  -secretArn string
        ARN of GitHub Runner Secret
  -version
        Show version of this app
```

## What does github-runner-utils do?

1. First retrieve the PEM stored in SSM Parameter Store and convert it to **JWT** for a Bearer Token which can be used to retreive the runner installation token in the Authorization header.

2. Then call GitHub API to get the App token, which is set on the GitHub app. Command line looks like:
    ```
    APP_TOKEN=$(curl --location --request POST "https://api.github.com/app/installations/${GH_APP_ID}/access_tokens" \
      --header "Authorization: Bearer ${JWT_TOKEN}" \
      --header 'Accept: application/vnd.github.v3+json' | jq -r '.token')
    ```

3. Then call GitHub API to get the runner registration token, which is a short lived token for (only) adding the runners. Command line looks like:
    ```
    REGO_TOKEN=$(curl --location --request POST 'https://api.github.com/orgs/${GH_ORG_NAME}/actions/runners/registration-token' \
      --header "Authorization: token ${APP_TOKEN}" \
      --header 'Accept: application/vnd.github.machine-man-preview+json' | jq -r '.token')
    ```

The runner registration token is needed to complete the runner installation.

For example, in your UserData,
```
cd /opt/github/actions-runner
./config.sh --url https://github.com/${GH_ORG_NAME} --token $REGO_TOKEN
./svc.sh install
./svc.sh start
```

## Build Binary

Tested with `go version go1.17.6 linux/amd64`

```
## Install and output binary
$ go build

## Run binary
$ ./github-runner-utils \
  -appId ${GH_APP_ID} \
  -installId "${GH_APP_ID}" \
  -orgName "${GH_ORG_NAME}" \
  -secretArn ${SECRET_MANAGER_SECRET_ARN_OF_GITHUB_PEM} \
  -iamRoleArn ${IAM_ROLE_ARN_ALLOW_READ_SECRET} \
  -awsRegion "ap-southeast-2"
```

Optionally override the `Version` string in source.
```
$ go build -v -ldflags="-X 'main.Version=v1.0.0'"
```

## Local Build and Run

```
## Update module: Output: go.mod being updated, go.sum being created/updated
$ go mod tidy

## Build and saves the compiled package in the local build cache. Output: ./github-runner-utils
$ go build

## Run locally
$ go run github-runner-utils.go \
  -appId ${GH_APP_ID} \
  -installId "${GH_APP_ID}" \
  -orgName "${GH_ORG_NAME}" \
  -secretArn ${SECRET_MANAGER_SECRET_ARN_OF_GITHUB_PEM} \
  -iamRoleArn ${IAM_ROLE_ARN_ALLOW_READ_SECRET} \
  -awsRegion "ap-southeast-2"
```

## Run unit tests
```
$ go test -v

# With coverage
$ go test -v -coverprofile=coverage.out
```

## Development Notes

1. Ensure your go.sum file is committed along with your go.mod file.
2. Default golang in Ubuntu 20 is 1.13. `github.com/golang-jwt/jwt` needs a newer Go version.
3. Standard "log" module does not have log level.
