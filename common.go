package awsauth

import (
	"bufio"
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type location struct {
	ec2     bool
	checked bool
}

var loc *location

// serviceAndRegion parsers a hostname to find out which ones it is.
// http://docs.aws.amazon.com/general/latest/gr/rande.html
func serviceAndRegion(host string) (service string, region string) {
	// These are the defaults if the hostname doesn't suggest something else
	region = "us-east-1"
	service = "s3"

	parts := strings.Split(host, ".")
	if len(parts) == 4 {
		// Either service.region.amazonaws.com or virtual-host.region.amazonaws.com
		if parts[1] == "s3" {
			service = "s3"
		} else if strings.HasPrefix(parts[1], "s3-") {
			region = region[3:]
			service = "s3"
		} else {
			service = parts[0]
			region = parts[1]
		}
	} else {
		// Either service.amazonaws.com or s3-region.amazonaws.com
		if strings.HasPrefix(parts[0], "s3-") {
			region = parts[0][3:]
		} else {
			service = parts[0]
		}
	}

	return
}

func checkKeys() {
	if Keys == nil {
		Keys = &Credentials{
			AccessKeyID:     os.Getenv(envAccessKeyID),
			SecretAccessKey: os.Getenv(envSecretAccessKey),
			SecurityToken:   os.Getenv(envSecurityToken),
		}
	}

	// If there is no Access Key and you are on EC2, get the key from the role
	if Keys.AccessKeyID == "" && onEC2() {
		Keys = getIAMRoleCredentials()
	}

	// If the key is expiring, get a new key
	if Keys.expired() && onEC2() {
		Keys = getIAMRoleCredentials()
	}
}

// onEC2 checks to see if the program is running on an EC2 instance.
// It does this by looking for the EC2 metadata service.
// This caches that information in a struct so that it doesn't waste time.
func onEC2() bool {
	if loc == nil {
		loc = &location{}
	}
	if !(loc.checked) {
		c, err := net.DialTimeout("tcp", "169.254.169.254:80", time.Second)

		if err != nil {
			loc.ec2 = false
		} else {
			c.Close()
			loc.ec2 = true
		}
		loc.checked = true
	}

	return loc.ec2
}

// getIAMRoleList gets a list of the roles that are available to this instance
func getIAMRoleList() []string {

	var roles []string
	url := "http://169.254.169.254/latest/meta-data/iam/security-credentials/"

	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return roles
	}

	resp, err := client.Do(req)

	if err != nil {
		return roles
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		roles = append(roles, scanner.Text())
	}
	return roles
}

func getIAMRoleCredentials() *Credentials {

	roles := getIAMRoleList()

	if len(roles) < 1 {
		return &Credentials{}
	}

	// Use the first role in the list
	role := roles[0]

	url := "http://169.254.169.254/latest/meta-data/iam/security-credentials/"

	// Create the full URL of the role
	var buffer bytes.Buffer
	buffer.WriteString(url)
	buffer.WriteString(role)
	roleurl := buffer.String()

	// Get the role
	rolereq, err := http.NewRequest("GET", roleurl, nil)

	if err != nil {
		return &Credentials{}
	}

	client := &http.Client{}
	roleresp, err := client.Do(rolereq)

	if err != nil {
		return &Credentials{}
	}
	defer roleresp.Body.Close()

	rolebuf := new(bytes.Buffer)
	rolebuf.ReadFrom(roleresp.Body)

	creds := Credentials{}

	err = json.Unmarshal(rolebuf.Bytes(), &creds)

	if err != nil {
		return &Credentials{}
	}

	return &creds

}

func augmentRequestQuery(req *http.Request, values url.Values) *http.Request {
	for key, arr := range req.URL.Query() {
		for _, val := range arr {
			values.Set(key, val)
		}
	}

	req.URL.RawQuery = values.Encode()

	return req
}

func hmacSHA256(key []byte, content string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(content))
	return mac.Sum(nil)
}

func hmacSHA1(key []byte, content string) []byte {
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(content))
	return mac.Sum(nil)
}

func hashSHA256(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func hashMD5(content []byte) string {
	h := md5.New()
	h.Write(content)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func readAndReplaceBody(req *http.Request) []byte {
	if req.Body == nil {
		return []byte{}
	}
	payload, _ := ioutil.ReadAll(req.Body)
	req.Body = ioutil.NopCloser(bytes.NewReader(payload))
	return payload
}

func concat(delim string, str ...string) string {
	return strings.Join(str, delim)
}

var now = func() time.Time {
	return time.Now().UTC()
}
