package utils

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

var (
	TestMode      = false
	SlackTestPort = ""
)

// DateTimeString - returns a string of  the current date and time
func DateTimeString() string {
	return fmt.Sprintf("%v", time.Now())
}

// func UpdateOpenAIConfig(c *OpenAIConfig, oa *OpenAIConfig) {
// Create a has of a string using the SHA256 algorithm
func HashString(s string) string {
	// use 	sha256 to hash a string
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// PrettyStructDebug - returns a pretty printed string of a struct
func PrettyStructDebug(v interface{}) string {
	jv, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		fmt.Printf("PrettyStructDebug - json.MarshalIndent err=%v\n", err)
	}
	return string(jv)
}

func PrintPrettyStructDebug(v interface{}) {
	jv, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		fmt.Printf("PrettyStructDebug - json.MarshalIndent err=%v\n", err)
	}
	fmt.Printf("\nDEBUG\n%v\n", string(jv))
}

func CheckEnvVars(envVars string) error {
	// Check that all the required environment variables are set
	// using the envVars const
	// returns an error if any are missing
	// otherwise returns nil
	for _, e := range strings.Split(envVars, ",") {
		if os.Getenv(e) == "" {
			return fmt.Errorf("checkEnvVars - Environment variable %s is not set", e)
		}
	}
	return nil
}

// write a function that returns a bool if it is passed a string of true or false
// if nether then return false
func StringToBool(s string) bool {
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	return false
}

func Prompt(f string) string {
	d, err := ioutil.ReadFile(f)
	if err != nil {
		log.Fatal(err)
	}

	return string(d)
}
