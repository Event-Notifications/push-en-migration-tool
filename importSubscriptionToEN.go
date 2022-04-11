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

func makeSubscribeCall(suburl string, device_id string, tag_name string, csvwriterF *csv.Writer, csvwriterS *csv.Writer) (string, error) {
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
		makeSubscribeCall(suburl, device_id, tag_name, csvwriterF, csvwriterS)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var strArr []string
	strArr = append(strArr, tag_name)
	strArr = append(strArr, device_id)

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		fmt.Println("Registered Subscription with response", string(body))
		_ = csvwriterS.Write(strArr)
	} else if resp.StatusCode == 409 {
		fmt.Println("Subscription already exists with DeviceID", device_id, tag_name, resp.StatusCode)
		_ = csvwriterS.Write(strArr)
		return "", err
	} else {
		fmt.Println("Failed Subscription with DeviceID", device_id, tag_name, resp.StatusCode)
		_ = csvwriterF.Write(strArr)
		return "", err
	}

	defer resp.Body.Close()

	return string(body), nil
}

func postDevice(enurl string, input string, csvwriterF *csv.Writer, csvwriterS *csv.Writer) (string, error) {

	inputSplit := strings.Split(input, ",")

	en_ios_sub_url := enurl + instanceID + "/destinations/" + iosDestinationID + "/tag_subscriptions"
	en_fcm_sub_url := enurl + instanceID + "/destinations/" + androidDestinationID + "/tag_subscriptions"

	makeSubscribeCall(en_fcm_sub_url, inputSplit[1], inputSplit[0], csvwriterF, csvwriterS)
	makeSubscribeCall(en_ios_sub_url, inputSplit[1], inputSplit[0], csvwriterF, csvwriterS)

	return "", nil
}

type result struct {
	bodyStr string
	err     error
}

func AsyncHTTP(enurl string, users []string, csvwriterF *csv.Writer, csvwriterS *csv.Writer) ([]string, error) {
	done := make(chan struct{})
	defer close(done)

	inputCh := streamInputs(done, users)

	var wg sync.WaitGroup

	wg.Add(GOROUTINE)

	resultCh := make(chan result)

	for i := 0; i < GOROUTINE; i++ {
		go func() {
			for input := range inputCh {
				bodyStr, err := postDevice(enurl, input, csvwriterF, csvwriterS)
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

	var regionMap = make(map[string]string)

	regionMap["stage"] = "https://us-south.imfpush.test.cloud.ibm.com/imfpush/v1/apps/"
	regionMap["dallas"] = "http://us-south.imfpush.cloud.ibm.com/imfpush/v1/apps/"
	regionMap["london"] = "https://eu-gb.imfpush.cloud.ibm.com/imfpush/v1/apps/"
	regionMap["sydney"] = "https://au-syd.imfpush.cloud.ibm.com/imfpush/v1/apps/"
	regionMap["frankfurt"] = "https://eu-de.imfpush.cloud.ibm.com/imfpush/v1/apps/"
	regionMap["washington"] = "https://us-east.imfpush.cloud.ibm.com/imfpush/v1/apps/"
	regionMap["tokyo"] = "https://jp-tok.imfpush.cloud.ibm.com/imfpush/v1/apps/"

	var enurl = regionMap[os.Getenv("EN_INSTANCE_REGION")]

	if enurl == "" {
		fmt.Println("Error processing request please check setEnv.sh and source it by adding region")
		return
	}

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

	csvFileFailed, err := os.Create("failed_subscription.csv")
	csvFileSucc, err := os.Create("migrated_subscription.csv")

	csvwriterFailed := csv.NewWriter(csvFileFailed)
	csvwriterSucc := csv.NewWriter(csvFileSucc)

	results, err := AsyncHTTP(enurl, subs, csvwriterFailed, csvwriterSucc)
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
