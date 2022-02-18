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

type GCPClient struct {
	service_account map[string]interface{} // from .json key exported from GCP service account
	access_token string
}

func NewGCPClient(service_account_json string) (*GCPClient, error) {
	client := &GCPClient{}

	// read key file
	jsonFile, err := os.Open(service_account_json)
	if err != nil {
		return nil, err
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(byteValue), &client.service_account)
	if err != nil {
		return nil, err
	}

	

	return client, nil
}

func (c *GCPClient) GetAccessToken() (string, error) {
	if c.access_token != "" {
		// TODO: refresh it if stale?
		return c.access_token, nil
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": c.service_account["client_email"],
		"scope": "https://www.googleapis.com/auth/compute",
		"aud": c.service_account["token_uri"],
		"exp": now.Add(time.Minute * 30).Unix(),
		"iat": now.Unix(),
	})

	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(c.service_account["private_key"].(string)))
	if err != nil {
		return "", err
	}

	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(c.service_account["token_uri"].(string),
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

	c.access_token = result["access_token"].(string)
	return c.access_token, nil
}

func (c *GCPClient) get(url string) (map[string]interface{}, error) {
	var result map[string]interface{}

	token, err := c.GetAccessToken()
	if err != nil {
		return result, err
	}

	url = fmt.Sprintf("%s?access_token=%s", url, token)
	resp, err := http.Get(url)
	if err != nil {
		return result, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return result, err
	}

	return result, nil
}

func (c *GCPClient) GcpListInstances() (map[string]interface{}, error) {
	return c.get("https://www.googleapis.com/compute/v1/projects/cs320-f21/zones/us-central1-a/instances")
}

func (c *GCPClient) GcpIPtoInstance() (map[string]string, error) {
	resp, err := c.GcpListInstances()
	if err != nil {
		return nil, err
	}

	lookup := map[string]string{}

	for _, item := range resp["items"].([]interface{}) {
		instance_name := item.(map[string]interface{})["name"].(string)
		interfaces := item.(map[string]interface{})["networkInterfaces"]
		for _, netif := range interfaces.([]interface{}) {
			confs := netif.(map[string]interface{})["accessConfigs"]
			for _, conf := range confs.([]interface{}) {
				iptmp := conf.(map[string]interface{})["natIP"]
				switch ip := iptmp.(type) {
				case string:
					lookup[ip] = instance_name
				}
			}
		}
	}

	return lookup, nil
}

func (c *GCPClient) GcpSnapshot() {
	token, err := c.GetAccessToken()
	if err != nil {
		panic(err)
	}

	// STEP 1: build body of REST request
	
	// TODO: take args from config (or better, read from service account somehow)
	args := GcpSnapshotArgs{
		Project: "cs320-f21",
		Region: "us-central1",
		Zone: "us-central1-a",
		Disk: "instance-2",
		SnapshotName: "test-snap",
	}
	temp := template.Must(template.New("gcp-launch").Parse(gcpSnapshotJSON))

	var payload bytes.Buffer
	if err := temp.Execute(&payload, args); err != nil {
		panic (err)
	}

	fmt.Printf("%s\n", string(payload.Bytes()))

	// STEP 3: Snapshot VM
	url := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/disks/%s/createSnapshot?access_token=%s",
		args.Project, args.Zone, args.Disk, token)
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

func (c *GCPClient) LaunchGCP() {
	token, err := c.GetAccessToken()
	if err != nil {
		panic(err)
	}

	// STEP 1: build body of REST request
	
	// TODO: take args from config (or better, read from service account somehow)
	args := GcpLaunchVmArgs{
		ServiceAccountEmail: c.service_account["client_email"].(string),
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
