package boss

import (
	"os"
	"io"
	"io/ioutil"
	"fmt"
	"time"
	"encoding/json"
	"github.com/golang-jwt/jwt"
	"net/http"
	"strings"
	"text/template"
	"bytes"
)

func getServiceAccount(service_account_json string) (map[string]interface{}, error) {
	var result map[string]interface{}
	jsonFile, err := os.Open(service_account_json)
	if err != nil {
		return result, err
	}
	defer jsonFile.Close()
	
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return result, err
	}

	err = json.Unmarshal([]byte(byteValue), &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func getGCPtoken(service_account map[string]interface{}) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": service_account["client_email"],
		"scope": "https://www.googleapis.com/auth/compute",
		"aud": service_account["token_uri"],
		"exp": now.Add(time.Minute * 30).Unix(),
		"iat": now.Unix(),
	})

	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(service_account["private_key"].(string)))
	if err != nil {
		return "", err
	}

	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(service_account["token_uri"].(string),
		"application/x-www-form-urlencoded",
		strings.NewReader("grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Ajwt-bearer&assertion="+tokenString))

	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return "", err
	}

	return result["access_token"].(string), nil
}

func main() {
	// TODO: take from config
	service_account, err := getServiceAccount("key.json")
	if err != nil {
		panic(err)
	}

	// STEP 1: build body of REST request
	
	// TODO: take args from config (or better, read from service account somehow)
	args := GcpLaunchVmArgs{
		ServiceAccountEmail: service_account["client_email"].(string),
		Project: "cs320-f21",
		Region: "us-central1",
		Zone: "us-central1-a",
		InstanceName: "instance-4",
		SourceImage: "projects/ubuntu-os-cloud/global/images/ubuntu-2004-focal-v20220204",
	}
	temp := template.Must(template.New("gcp-launch").Parse(gcpLaunchVmJSON))

	var payload bytes.Buffer
	if err := temp.Execute(&payload, args); err != nil {
		panic (err)
	}

	fmt.Printf("%s\n", string(payload.Bytes()))

	// STEP 2: get token for request

	// TODO: re-use this and renew before it expires
	token, err := getGCPtoken(service_account)
	if err != nil {
		panic(err)
	}

	// STEP 3: launch VM!
	url := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/instances?access_token=%s",
		args.Project, args.Zone, token)
	fmt.Printf("%s\n", url)

	resp, err := http.Post(url, "application/json", bytes.NewReader(payload.Bytes()))
	if err != nil {
		panic (err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%V\n", string(body))
}
