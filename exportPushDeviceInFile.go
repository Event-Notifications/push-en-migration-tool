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
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Response struct {
	PageInfo struct {
		TotalCount int    `json:"totalCount"`
		Next       string `json:"next"`
	} `json:"pageInfo"`

	Devices []struct {
		DeviceID string `json:"deviceId"`
		UserID   string `json:"userId"`
		Token    string `json:"token"`
		Platform string `json:"platform"`
	} `json:"devices"`
}

type IAMStruct struct {
	AccessToken string `json:"access_token"`
}

var instanceID = os.Getenv("PUSH_INSTANCE_ID")
var authorization = ""
var apiKey = os.Getenv("PUSH_APIKEY")

func getToken() {
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
		fmt.Println("Failed to get authorization token: ", err)
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result IAMStruct
	if err := json.Unmarshal(body, &result); err != nil { // Parse []byte to go struct pointer
		fmt.Println("Failed to read response body", body, err)
	}

	authorization = result.AccessToken

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

	var pushurl = regionMap[os.Getenv("PUSH_INSTANCE_REGION")]

	if pushurl == "" {
		fmt.Println("Error processing request please check setEnv.sh and source it by adding region")
		return
	}

	api := "/devices?expand=true&offset=0&size=500"

	csvFile, err := os.Create("devices.csv")
	csvwriter := csv.NewWriter(csvFile)
	if err != nil {
		log.Fatalf("Failed creating devices file: %s", err)
	}

	getDevice(pushurl+instanceID+api, csvwriter)

	csvwriter.Flush()
	csvFile.Close()

}

func getDevice(pushdeviceurl string, csvwriter *csv.Writer) error {

	if pushdeviceurl == "" {
		fmt.Println("Finished getting device")
		return nil
	}

	client := &http.Client{}

	req, _ := http.NewRequest("GET", pushdeviceurl, nil)

	req.Header.Add("Authorization", authorization)

	response, err := client.Do(req)

	if err != nil {
		fmt.Println("Error processing request please check setEnv.sh and source it", err)
		return err
	}
	body, _ := ioutil.ReadAll(response.Body)

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		log.Printf("Error decoding response: %v", err)
		if e, ok := err.(*json.SyntaxError); ok {
			log.Printf("Syntax error at byte offset %d", e.Offset)
		}
		log.Printf("Response: %q", body)
		return err
	}

	if response.StatusCode == 401 {
		getToken()
		getDevice(pushdeviceurl, csvwriter)
	}

	fmt.Println("Getting device with push device url", result.PageInfo.Next)

	for _, device := range result.Devices {
		var strArr []string
		strArr = append(strArr, device.DeviceID)
		strArr = append(strArr, device.UserID)
		strArr = append(strArr, device.Token)
		strArr = append(strArr, device.Platform)
		_ = csvwriter.Write(strArr)
	}

	defer response.Body.Close()

	getDevice(result.PageInfo.Next, csvwriter)

	return nil

}
