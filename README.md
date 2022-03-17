
# Push to EN Data Migration Tool

This migration tool helps migrate Push (IOS and Android) devices and subscriptions from Push Notifications Instance to Event Notifications Instance

## Requirements

This module requires the following modules:

* [Go](https://go.dev/doc/install)

Ability to run bash scripts

## Prerequisites

#### Step 1 - Get Push Notifications Credentials

Get service credentials of your Push Notifications Instance  

- Push Instance Region API URL
- Push Instance ID
- Push APIKey
- Push Client Secret 

#### Step 2 - Create APNS Destinations in Event Notifications

https://cloud.ibm.com/docs/event-notifications?topic=event-notifications-en-push-apns

#### Step 3 - Create FCM Destinations in Event Notifications

https://cloud.ibm.com/docs/event-notifications?topic=event-notifications-en-create-send

#### Step 4 - Get Event Notifications Credentials

Get following details from your Event Notifications Instance

- EN Instance Region API URL
- EN Instance ID 
- EN APIkey
- EN APNS Destination ID generated at Step 2
- EN Android Destination ID generated at Step 3



## Usage

Follow these steps to migrate all IOS and Android devices from your push instance to event notification instance 


#### Step 1 - Source Credentials

Fill all your details from prerequisite steps in to file **setEnv.sh** and source it using the command ```source setEnv.sh```

#### Step 2 - Export Device from Push Instance

Run command ```go run exportPushDeviceInFile.go 2>&1 | tee logExportDevice.txt &``` , this will retrieve all devices from push instance to a file named **devices.csv**


#### Step 3 - Export Subscriptions from Push Instance

Run command ```go run exportPushSubscriptionInFile.go 2>&1 | tee logExportSubscription.txt &```, this will retrieve all subscriptions from push instance to a file named **subscriptions.csv**

#### Step 4 - Import Devices to EN Instance

Run command ```go run importPushDevicesToEN.go 2>&1 | tee logImportDevice.txt &```, this will register all devices from push to EN destinations IOS and Android respectively. 

#### Step 5 - Import Subscriptions to EN Instance

Run command ```go run importSubscriptionToEN.go  2>&1 | tee logImportDevice.txt &```, this will subscribe tags from push to en . 



# NOTE

- All commands run in background and stores logs in a file
- Any failures in request will be saved in **failed_devices.csv** and **failed_subscription.csv**. After tool is finished these can be tried again by renaming to **devices.csv** and **subscription.csv** respectively
