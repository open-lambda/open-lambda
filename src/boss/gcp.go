package boss

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/golang-jwt/jwt"
)

type GCPClient struct {
	service_account map[string]any // from .json key exported from GCP service account
	access_token    string
}

func GCPBossTest() {
	fmt.Printf("STEP 0: check SSH setup\n")
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	tmp, err := os.ReadFile(filepath.Join(home, ".ssh", "id_rsa.pub"))
	if err != nil {
		panic(err)
	}
	pub := strings.TrimSpace(string(tmp))

	tmp, err = os.ReadFile(filepath.Join(home, ".ssh", "authorized_keys"))
	if err != nil {
		panic(err)
	}
	authorized := strings.Split(string(tmp), "\n")

	matches := false
	for _, v := range authorized {
		if strings.TrimSpace(v) == pub {
			matches = true
			break
		}
	}

	if !matches {
		panic(fmt.Errorf("could not find id_rsa.pub in authorized_keys, consider running: cat ~/.ssh/id_rsa.pub >> ~/.ssh/authorized_keys "))
	}

	fmt.Printf("STEP 1: get access token\n")
	client, err := NewGCPClient("key.json")
	if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 2: lookup instance from IP address\n")
	instance, err := client.GcpInstanceName()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Instance: %s\n", instance)

	fmt.Printf("STEP 3: take crash-consistent snapshot of instance\n")
	disk := instance // assume GCP disk name is same as instance name
	resp, err := client.Wait(client.GcpSnapshot(disk))

	fmt.Println(resp)
	if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 4: create new VM from snapshot\n")
	resp, err = client.Wait(client.LaunchGCP("test-snap", "test-vm"))
	fmt.Println(resp)
	if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 5: start worker\n")
	err = client.StartRemoteWorker()
	if err != nil {
		panic(err)
	}
}

func NewGCPClient(service_account_json string) (*GCPClient, error) {
	client := &GCPClient{}

	// read key file
	jsonFile, err := os.Open(service_account_json)
	if err != nil {
		fmt.Printf("To get a .json KEY for a service account, go to https://console.cloud.google.com/iam-admin/serviceaccounts")
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

func (c *GCPClient) StartRemoteWorker() error {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	lookup, err := c.GcpInstancetoIP()
	if err != nil {
		panic(err)
	}
	ip, ok := lookup["instance-4"] // TODO
	if !ok {
		fmt.Println(lookup)
		panic(fmt.Errorf("could not find IP for instance"))
	}

	cmd := fmt.Sprintf("cd %s; %s", cwd, "./ol worker --detach")

	tries := 10
	for tries > 0 {
		sshcmd := exec.Command("ssh", user.Username+"@"+ip, "-o", "StrictHostKeyChecking=no", "-C", cmd)
		stdoutStderr, err := sshcmd.CombinedOutput()
		fmt.Printf("%s\n", stdoutStderr)
		if err == nil {
			break
		}
		tries -= 1
		if tries == 0 {
			fmt.Println(sshcmd.String())
			panic(err)
		}
		time.Sleep(5 * time.Second)
	}

	return nil
}

func (c *GCPClient) GetAccessToken() (string, error) {
	if c.access_token != "" {
		// TODO: refresh it if stale?
		return c.access_token, nil
	}

	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss":   c.service_account["client_email"],
		"scope": "https://www.googleapis.com/auth/compute",
		"aud":   c.service_account["token_uri"],
		"exp":   now.Add(time.Minute * 30).Unix(),
		"iat":   now.Unix(),
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

	var result map[string]any
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return "", err
	}

	c.access_token = result["access_token"].(string)
	return c.access_token, nil
}

func (c *GCPClient) get(url string) (rv map[string]any, err error) {
	var result map[string]any

	defer func() {
		if err != nil {
			err = fmt.Errorf("POST to %s failed: %s", url, err.Error())
		}
	}()

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

func (c *GCPClient) post(url string, payload bytes.Buffer) (rv map[string]any, err error) {
	var result map[string]any

	defer func() {
		if err != nil {
			err = fmt.Errorf("POST to %s failed: %s", url, err.Error())
		}
	}()

	token, err := c.GetAccessToken()
	if err != nil {
		return result, err
	}

	url = fmt.Sprintf("%s?access_token=%s", url, token)
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload.Bytes()))
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

func (c *GCPClient) GcpListInstances() (map[string]any, error) {
	return c.get("https://www.googleapis.com/compute/v1/projects/cs320-f21/zones/us-central1-a/instances")
}

func (c *GCPClient) GcpIPtoInstance() (map[string]string, error) {
	resp, err := c.GcpListInstances()
	if err != nil {
		return nil, err
	}

	lookup := map[string]string{}

	for _, item := range resp["items"].([]any) {
		instance_name := item.(map[string]any)["name"].(string)
		interfaces := item.(map[string]any)["networkInterfaces"]
		for _, netif := range interfaces.([]any) {
			ip := netif.(map[string]any)["networkIP"].(string)
			lookup[ip] = instance_name

			confs := netif.(map[string]any)["accessConfigs"]
			for _, conf := range confs.([]any) {
				iptmp := conf.(map[string]any)["natIP"]
				switch ip := iptmp.(type) {
				case string:
					lookup[ip] = instance_name
				}
			}
		}
	}

	return lookup, nil
}

func (c *GCPClient) GcpInstancetoIP() (map[string]string, error) {
	lookup1, err := c.GcpIPtoInstance()
	if err != nil {
		return nil, err
	}

	lookup2 := map[string]string{}
	for k, v := range lookup1 {
		lookup2[v] = k
	}

	return lookup2, nil
}

// https://stackoverflow.com/questions/23558425/how-do-i-get-the-local-ip-address-in-go
func getOutboundIP() (string, error) {
	// we might be behind a
	conn, err := net.Dial("udp", "8.8.8.8:80") // TODO: lookup DNS server from config
	if err != nil {
		return "", err
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP.String(), nil
}

func (c *GCPClient) GcpInstanceName() (string, error) {
	lookup, err := c.GcpIPtoInstance()
	if err != nil {
		return "", nil
	}

	ip, err := getOutboundIP()
	if err != nil {
		return "", nil
	}

	instance, ok := lookup[ip]
	if !ok {
		return "", fmt.Errorf("could not find GCP instance for %s", ip)
	}
	return instance, nil
}

func (c *GCPClient) Wait(resp1 map[string]any, err1 error) (resp2 map[string]any, err2 error) {
	if err1 != nil {
		return nil, fmt.Errorf("cannot Wait on on failed call: %s", err1.Error())
	}

	selfLink, ok := resp1["selfLink"]
	if !ok {
		return resp1, fmt.Errorf("GCP REST operation did not succeed")
	}

	poll_url := selfLink.(string) // TODO: + "/wait"

	for i := 0; i < 30; i++ {
		resp2, err2 = c.get(poll_url)
		if err2 != nil {
			return nil, err2
		}

		fmt.Println("POLLING", resp2)
		fmt.Println()

		if resp2["status"].(string) != "RUNNING" {
			return resp2, nil
		}

		time.Sleep(10 * time.Second)
	}

	return resp2, fmt.Errorf("Wait: operation timed out")
}

func (c *GCPClient) GcpSnapshot(disk string) (map[string]any, error) {
	// TODO: take args from config (or better, read from service account somehow)
	args := GcpSnapshotArgs{
		Project:      "cs320-f21",
		Region:       "us-central1",
		Zone:         "us-central1-a",
		Disk:         disk,
		SnapshotName: "test-snap",
	}

	url := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/disks/%s/createSnapshot",
		args.Project, args.Zone, args.Disk)

	var payload bytes.Buffer
	temp := template.Must(template.New("gcp-snap").Parse(gcpSnapshotJSON))
	if err := temp.Execute(&payload, args); err != nil {
		panic(err)
	}

	return c.post(url, payload)
}

func (c *GCPClient) LaunchGCP(SnapshotName string, VMName string) (map[string]any, error) {
	// TODO: take args from config (or better, read from service account somehow)
	args := GcpLaunchVmArgs{
		ServiceAccountEmail: c.service_account["client_email"].(string),
		Project:             "cs320-f21",
		Region:              "us-central1",
		Zone:                "us-central1-a",
		InstanceName:        VMName,
		//SourceImage: "projects/ubuntu-os-cloud/global/images/ubuntu-2004-focal-v20220204",
		SnapshotName: SnapshotName,
	}

	url := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/instances",
		args.Project, args.Zone)

	var payload bytes.Buffer
	temp := template.Must(template.New("gcp-launch").Parse(gcpLaunchVmJSON))
	if err := temp.Execute(&payload, args); err != nil {
		panic(err)
	}

	return c.post(url, payload)
}
