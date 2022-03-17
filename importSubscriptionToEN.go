/**
 * (C) Copyright IBM Corp. 2022.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type IAMStruct struct {
	AccessToken string `json:"access_token"`
}

var enurl = os.Getenv("EN_URL")
var instanceID = os.Getenv("EN_INSTANCE_ID")
var iosDestinationID = os.Getenv("EN_IOS_DESTINATION_ID")
var androidDestinationID = os.Getenv("EN_ANDROID_DESTINATION_ID")
var apiKey = os.Getenv("EN_APIKEY")
var authorization = ""

const GOROUTINE = 15

func getToken() error {
	client := &http.Client{}
	iamURL := "https://iam.cloud.ibm.com/identity/token"

	data := url.Values{}
	data.Set("grant_type", "urn:ibm:params:oauth:grant-type:apikey")
	data.Set("apikey", apiKey)

	req, _ := http.NewRequest("POST", iamURL, strings.NewReader(data.Encode()))

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Error processing request please check setEnv.sh and source it", err)
		return err
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result IAMStruct
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error decoding response: %v", err)
		if e, ok := err.(*json.SyntaxError); ok {
			log.Printf("Syntax error at byte offset %d", e.Offset)
		}
		log.Printf("Response: %q", body)
		return err
	}

	authorization = result.AccessToken

	return nil

}

func streamInputs(done <-chan struct{}, inputs []string) <-chan string {
	inputCh := make(chan string)
	go func() {
		defer close(inputCh)
		for _, input := range inputs {
			select {
			case inputCh <- input:
			case <-done:
				break
			}
		}
	}()
	return inputCh
}

func makeSubscribeCall(suburl string, device_id string, tag_name string, csvwriter *csv.Writer) (string, error) {
	client := &http.Client{}

	postBody, _ := json.Marshal(map[string]string{
		"device_id": device_id,
		"tag_name":  tag_name,
	})

	reqBody := bytes.NewBuffer(postBody)

	req, _ := http.NewRequest("POST", suburl, reqBody)

	req.Header.Add("Authorization", "Bearer "+authorization)
	req.Header.Add("Content-Type", "application/json")

	resp, err := client.Do(req)

	if resp.StatusCode == 401 {
		getToken()
		makeSubscribeCall(suburl, device_id, tag_name, csvwriter)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		fmt.Println("Registered Subscription with response", string(body))
	} else {
		fmt.Println("Failed Device with DeviceID", device_id, resp.StatusCode)
		var strArr []string
		strArr = append(strArr, tag_name)
		strArr = append(strArr, device_id)

		_ = csvwriter.Write(strArr)
		return "", err
	}

	defer resp.Body.Close()

	return string(body), nil
}

func postDevice(input string, csvwriter *csv.Writer) (string, error) {

	inputSplit := strings.Split(input, ",")

	en_ios_sub_url := enurl + instanceID + "/destinations/" + iosDestinationID + "/tag_subscriptions"
	en_fcm_sub_url := enurl + instanceID + "/destinations/" + androidDestinationID + "/tag_subscriptions"

	makeSubscribeCall(en_fcm_sub_url, inputSplit[1], inputSplit[0], csvwriter)
	makeSubscribeCall(en_ios_sub_url, inputSplit[1], inputSplit[0], csvwriter)

	return "", nil
}

type result struct {
	bodyStr string
	err     error
}

func AsyncHTTP(users []string, csvwriter *csv.Writer) ([]string, error) {
	done := make(chan struct{})
	defer close(done)

	inputCh := streamInputs(done, users)

	var wg sync.WaitGroup

	wg.Add(GOROUTINE)

	resultCh := make(chan result)

	for i := 0; i < GOROUTINE; i++ {
		go func() {
			for input := range inputCh {
				bodyStr, err := postDevice(input, csvwriter)
				resultCh <- result{bodyStr, err}
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := []string{}
	for result := range resultCh {
		if result.err != nil {
			return nil, result.err
		}

		results = append(results, result.bodyStr)
	}

	return results, nil
}

func main() {
	getToken()
	subs := []string{}

	file, err := os.Open("subscription.csv")
	if err != nil {
		fmt.Println(err)
	}
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()

	if err != nil {
		fmt.Println(err)
	}

	for _, record := range records {
		tagName := record[0]
		deviceID := record[1]

		row := tagName + "," + deviceID

		subs = append(subs, row)
	}

	start := time.Now()

	csvFile, err := os.Create("failed_subscription.csv")
	csvwriter := csv.NewWriter(csvFile)
	if err != nil {
		log.Fatalf("Failed creating devices file: %s", err)
	}

	results, err := AsyncHTTP(subs, csvwriter)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, result := range results {
		fmt.Println(result)
	}
	csvwriter.Flush()
	csvFile.Close()

	fmt.Println("finished in ", time.Since(start))
}
