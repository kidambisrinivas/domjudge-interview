package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/crypto/bcrypt"
)

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ!@#$%^&*-+;()"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// Generate random string bytes
func RandStringBytesMaskImprSrcUnsafe(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

// Get password hash
func GetPasswordHash(pswd string) (hash string, err error) {
	res, err := bcrypt.GenerateFromPassword([]byte(pswd), 10)
	hash = string(res)
	return hash, err
}

// Send HTML template based email using sendwithus service (assumes that a template has been
// created using sendwithus service online using their dashboard)
func SendEmailUsingSendwithus(to, toName, from, fromName, replyTo, cc, bcc, apiKey, templateId string, templateData *ContestWelcomeEmail) (body string, err error) {
	if to == "" || toName == "" || from == "" || fromName == "" || templateId == "" {
		return "", PrintErr("SENDWITHUS_BADINPUT", fmt.Sprintf("Mandatory params not sent in (to: %s, toName: %s, from: %s, fromName: %s, templateId: %s)", to, toName, from, fromName, templateId))
	}

	// Construct sendwithus payload
	sendWithUsPayload := map[string]interface{}{
		"template":      fmt.Sprintf("%s", templateId),
		"template_data": templateData,
		"recipient": map[string]interface{}{
			"address": to,
			"name":    toName,
		},
		"sender": map[string]interface{}{
			"address":  from,
			"name":     fromName,
			"reply_to": replyTo,
		},
	}
	// files, filesPresent := templateData["files"]
	// if filesPresent && files != nil {
	// 	sendWithUsPayload["files"] = templateData["files"]
	// 	delete(templateData, "files")
	// }

	copyMails := []string{"cc", "bcc"}
	for _, addressType := range copyMails {
		addressesStr := ""
		if addressType == "cc" {
			addressesStr = cc
		} else {
			addressesStr = bcc
		}
		if addressesStr != "" {
			addresses := strings.Split(addressesStr, ",")
			addressesObjs := []map[string]interface{}{}
			for _, emailId := range addresses {
				addressObj := map[string]interface{}{"address": emailId}
				addressesObjs = append(addressesObjs, addressObj)
			}
			sendWithUsPayload[addressType] = addressesObjs
		}
	}

	apiBase := fmt.Sprintf("https://%s:@api.sendwithus.com/api/v1/", apiKey)
	url := fmt.Sprintf("%s%s", apiBase, "send")
	jsonStr, _ := json.MarshalIndent(sendWithUsPayload, "", "  ")
	log.Printf("SENDWITHUS_EMAIL: (%s) Req %v\n", url, string(jsonStr))
	_, bodyBytes, err := RequestUrl("POST", url, sendWithUsPayload, "", 120)
	if err != nil {
		return "", PrintErr("SENDWITHUS_ERR", fmt.Sprintf("failed to %s %s (%s): %v", "POST", url, string(jsonStr), err))
	}
	body = string(bodyBytes)
	log.Printf("SENDWITHUS_RESP: POST %s -> %s\n", url, body)
	return body, nil
}

// Encodes the object and make PUT/POST request for give url
func RequestUrl(reqType string, url string, inputData interface{}, userName string, timeout int) (status int, body []byte, err error) {
	if timeout == 0 {
		timeout = 60
	}

	b, err := json.Marshal(inputData)
	if err != nil {
		return status, body, err
	}
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}
	req, err := http.NewRequest(reqType, url, bytes.NewBuffer(b))
	if err != nil {
		return status, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if userName != "" {
		req.Header.Set("x-gateway-user-id", userName)
	}
	res, err := client.Do(req)
	if err != nil {
		return status, nil, err
	}
	status = res.StatusCode
	defer res.Body.Close()
	resBody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return status, nil, err
	}

	return status, resBody, nil
}

func GetStringKey(m map[string]interface{}, k string) (vstr string, ok bool) {
	v, ok := m[k]
	if ok {
		vstr, ok = v.(string)
		return vstr, ok
	}
	return vstr, ok
}
