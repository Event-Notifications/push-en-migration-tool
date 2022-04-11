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
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type Response struct {
	PageInfo struct {
		TotalCount int    `json:"totalCount"`
		Next       string `json:"next"`
	} `json:"pageInfo"`

	Subscriptions []struct {
		TagName  string `json:"tagName"`
		DeviceID string `json:"deviceId"`
	} `json:"subscriptions"`
}

var pushurl = os.Getenv("PUSH_URL")
var instanceID = os.Getenv("PUSH_INSTANCE_ID")

func main() {

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

	api := "/subscriptions?expand=true&offset=0&size=500"
	csvFile, err := os.Create("subscription.csv")
	csvwriter := csv.NewWriter(csvFile)
	if err != nil {
		log.Fatalf("Failed creating subscription file: %s", err)
	}

	getDevice(pushurl+instanceID+api, csvwriter)

	csvwriter.Flush()
	csvFile.Close()

}

func getDevice(url string, csvwriter *csv.Writer) error {

	if url == "" {
		fmt.Printf("Finished getting subscriptions")
		return nil
	}

	client := &http.Client{}

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("clientSecret", os.Getenv("PUSH_CLIENT_SECRET"))

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

	fmt.Println("Getting Subscription from url ", result.PageInfo.Next)

	for _, sub := range result.Subscriptions {
		var strArr []string

		if sub.TagName == "Push.ALL" {
			continue
		}

		strArr = append(strArr, sub.TagName)
		strArr = append(strArr, sub.DeviceID)

		_ = csvwriter.Write(strArr)
	}

	defer response.Body.Close()

	getDevice(result.PageInfo.Next, csvwriter)
	return nil
}
