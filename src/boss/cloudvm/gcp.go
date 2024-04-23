package cloudvm

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

type GcpClient struct {
	service_account map[string]any // from .json key exported from Gcp service account
	access_token    string
}

func GcpBossTest() {
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
	client, err := NewGcpClient("key.json")
	if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 1a: lookup region and zone from metadata server\n")
	region, zone, err := client.GcpProjectZone()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Region: %s\nZone: %s\n", region, zone)

	fmt.Printf("STEP 2: lookup instance from IP address\n")
	instance, err := client.GcpInstanceName()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Instance: %s\n", instance)

	fmt.Printf("STEP 3: take crash-consistent snapshot of instance\n")
	disk := instance // assume Gcp disk name is same as instance name
	start := time.Now()
	resp, err := client.Wait(client.GcpSnapshot(disk, "test-snap"))
	snapshot_time := time.Since(start)

	fmt.Println(resp)
	if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 4: create new VM from snapshot\n")
	start = time.Now()
	resp, err = client.Wait(client.LaunchGcp("test-snap", "test-vm"))
	clone_time := time.Since(start)
	if err != nil && resp["error"].(map[string]any)["code"] != "409" { // continue if instance already exists error
		fmt.Printf("instance alreay exists!\n")
		client.startGcpInstance("test-vm")
	} else if err != nil {
		panic(err)
	}

	fmt.Printf("snapshot time: %d\n", snapshot_time.Milliseconds())
	fmt.Printf("clone time: %d\n", clone_time.Milliseconds())

	fmt.Printf("STEP 5: start worker\n")
	err = client.RunComandWorker("test-vm", "./ol worker --detach")
	if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 6: stop instance\n")
	resp, err = client.Wait(client.stopGcpInstance("test-vm"))
	if err != nil {
		panic(err)
	}

	fmt.Printf("STEP 7: delete instance\n")
	resp, err = client.deleteGcpInstance("test-vm")
	if err != nil {
		panic(err)
	}
	fmt.Printf("Test Succeeded!\n")
}

func NewGcpClient(service_account_json string) (*GcpClient, error) {
	client := &GcpClient{}

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

func (c *GcpClient) RunComandWorker(vmName string, command string) error {
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

	ip, ok := lookup[vmName]
	if !ok {
		fmt.Println(lookup)
		panic(fmt.Errorf("could not find IP for instance"))
	}

	cmd := fmt.Sprintf("cd %s; %s", cwd, command)

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

func (c *GcpClient) GetAccessToken() (string, error) {
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

func (c *GcpClient) get(url string) (rv map[string]any, err error) {
	var result map[string]any

	defer func() {
		if err != nil {
			err = fmt.Errorf("GET to %s failed: %s", url, err.Error())
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

func (c *GcpClient) post(url string, payload bytes.Buffer) (rv map[string]any, err error) {
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

func (c *GcpClient) delete(url string) (rv map[string]any, err error) {
	var result map[string]any

	defer func() {
		if err != nil {
			err = fmt.Errorf("DELETE to %s failed: %s", url, err.Error())
		}
	}()

	token, err := c.GetAccessToken()
	if err != nil {
		return result, err
	}

	url = fmt.Sprintf("%s?access_token=%s", url, token)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return result, err
	}

	req.Header.Set("Content-Type", "application/json")
	client := http.Client{}
	resp, err := client.Do(req)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	if err := json.Unmarshal([]byte(body), &result); err != nil {
		return result, err
	}

	return result, nil
}

func (c *GcpClient) GcpProjectZone() (string, string, error) {
	url := fmt.Sprintf("http://metadata.google.internal/computeMetadata/v1/instance/zone")

	token, err := c.GetAccessToken()
	if err != nil {
		return "", "", err
	}

	url = fmt.Sprintf("%s?access_token=%s", url, token)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Metadata-Flavor", "Google")
	client := http.Client{}
	resp, err := client.Do(req)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	subs := strings.Split(string(body), "/")
	zone := subs[len(subs)-1]
	region := zone[:len(zone)-2]

	c.service_account["region"] = region
	c.service_account["zone"] = zone

	return region, zone, nil
}

func (c *GcpClient) GcpListInstances() (map[string]any, error) {
	url := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/instances", c.service_account["project_id"], c.service_account["zone"])
	return c.get(url)
}

func (c *GcpClient) GcpIPtoInstance() (map[string]string, error) {
	resp, err := c.GcpListInstances()
	if err != nil {
		return nil, err
	}

	lookup := map[string]string{}
	for _, item := range resp["items"].([]any) {
		instance_name := item.(map[string]any)["name"].(string)
		interfaces := item.(map[string]any)["networkInterfaces"]
		for _, netif := range interfaces.([]any) {
			ip := netif.(map[string]any)["networkIP"].(string) // internal ip
			lookup[ip] = instance_name
		}
	}

	return lookup, nil
}

func (c *GcpClient) GcpInstancetoIP() (map[string]string, error) {
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

func (c *GcpClient) GcpInstanceName() (string, error) {
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
		return "", fmt.Errorf("could not find Gcp instance for %s", ip)
	}
	return instance, nil
}

func (c *GcpClient) Wait(resp1 map[string]any, err1 error) (resp2 map[string]any, err2 error) {
	if err1 != nil {
		return nil, fmt.Errorf("cannot Wait on on failed call: %s", err1.Error())
	}

	selfLink, ok := resp1["selfLink"]
	if !ok {
		return resp1, fmt.Errorf("Gcp REST operation did not succeed")
	}

	poll_url := selfLink.(string) // TODO: + "/wait"

	for i := 0; i < 30; i++ {
		resp2, err2 = c.get(poll_url)
		if err2 != nil {
			return nil, err2
		}

		if resp2["status"].(string) != "RUNNING" {
			return resp2, nil
		}

		time.Sleep(10 * time.Second)
	}

	return resp2, fmt.Errorf("Wait: operation timed out")
}

func (c *GcpClient) GcpSnapshot(disk string, snapshot_name string) (map[string]any, error) {
	args := GcpSnapshotArgs{
		Project:      c.service_account["project_id"].(string),
		Region:       c.service_account["region"].(string),
		Zone:         c.service_account["zone"].(string),
		Disk:         disk,
		SnapshotName: snapshot_name,
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

func (c *GcpClient) LaunchGcp(snapshotName string, vmName string) (map[string]any, error) {
	args := GcpLaunchVmArgs{
		ServiceAccountEmail: c.service_account["client_email"].(string),
		Project:             c.service_account["project_id"].(string),
		Region:              c.service_account["region"].(string),
		Zone:                c.service_account["zone"].(string),
		InstanceName:        vmName,
		// SourceImage: "projects/ubuntu-os-cloud/global/images/ubuntu-2004-focal-v20220204",
		SnapshotName:        snapshotName,
		DiskSizeGb:          GcpConf.DiskSizeGb,
		MachineType:         GcpConf.MachineType,
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

// start existing instance
func (c *GcpClient) startGcpInstance(vmName string) (map[string]any, error) {
	url := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s/start",
		c.service_account["project_id"].(string),
		c.service_account["zone"].(string),
		vmName)

	var payload bytes.Buffer

	return c.post(url, payload)
}

func (c *GcpClient) stopGcpInstance(vmName string) (map[string]any, error) {
	url := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s/stop",
		c.service_account["project_id"].(string),
		c.service_account["zone"].(string),
		vmName)

	var payload bytes.Buffer

	return c.post(url, payload)
}

func (c *GcpClient) deleteGcpInstance(vmName string) (map[string]any, error) {
	url := fmt.Sprintf("https://www.googleapis.com/compute/v1/projects/%s/zones/%s/instances/%s",
		c.service_account["project_id"].(string),
		c.service_account["zone"].(string),
		vmName)

	return c.delete(url)
}
