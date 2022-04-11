
# Push to EN Data Migration Tool

This migration tool helps migrate Push (IOS and Android) devices and subscriptions from Push Notifications Instance to Event Notifications Instance

## Requirements

This module requires the following modules:

* [Go](https://go.dev/doc/install)

Ability to run bash scripts

## Prerequisites

#### Step 1 - Get Push Notifications Credentials

Get service credentials of your Push Notifications Instance  

- Push Instance Region - dallas/london/sydney/frankfurt/washington/tokyo/stage
- Push Instance ID
- Push APIKey
- Push Client Secret 

#### Step 2 - Create APNS Destinations in Event Notifications

https://cloud.ibm.com/docs/event-notifications?topic=event-notifications-en-push-apns

#### Step 3 - Create FCM Destinations in Event Notifications

https://cloud.ibm.com/docs/event-notifications?topic=event-notifications-en-create-send

#### Step 4 - Get Event Notifications Credentials

Get following details from your Event Notifications Instance

- EN Instance Region - dallas/london/frankfurt/sydney/stage
- EN Instance ID 
- EN APIkey
- EN APNS Destination ID generated at Step 2
- EN Android Destination ID generated at Step 3



## Usage

Follow these steps to migrate all IOS and Android devices from your push instance to event notification instance. During this migration process it is recommened to stop all the send notificaitons calls to IBM Push Notifications service instance.

#### Step 1 - Source Credentials

Fill all your details from prerequisite steps in to file **setEnv.sh** and source it using the command ```source setEnv.sh```

#### Step 2 - Export Device from Push Instance

Run command ```go run exportPushDeviceInFile.go 2>&1 | tee logExportDevice.txt &``` , this will retrieve all devices from push instance to a file named **devices.csv**


#### Step 3 - Export Subscriptions from Push Instance

Run command ```go run exportPushSubscriptionInFile.go 2>&1 | tee logExportSubscription.txt &```, this will retrieve all subscriptions from push instance to a file named **subscription.csv**

#### Step 4 - Import Devices to EN Instance

Run command ```go run importPushDevicesToEN.go 2>&1 | tee logImportDevice.txt &```, this will register all devices from push to EN destinations IOS and Android respectively. 

#### Step 5 - Import Subscriptions to EN Instance

Run command ```go run importSubscriptionToEN.go  2>&1 | tee logImportDevice.txt &```, this will subscribe tags from push to en . 


# NOTE

- All commands run in background and stores logs in a file
- Successful migrated requests will be saved in **migrated_devices.csv** and **migrated_subscription.csv**. Do not delete these files.
- Any failures in request will be saved in **failed_devices.csv**  and **failed_subscription.csv**. This is only for information and its of no use. Can be deleted.


After tool is finished failed requests can be tried again by running these command 

``` grep -vxFf migrated_devices.csv devices.csv > devices_new.csv```
``` grep -vxFf migrated_subscription.csv subscription.csv > subscription_new.csv```

Make a backup of olde files and rename devices_new to devices and subscription_new to subscription

Above commands

# IMPORTANT
  
  From time to time the Push providers (like FCM) will expire the existing push device token and provide a new push device token. The Event Notifications service client SDK sends this new token to your service instance destination. A valid token is required to send push notifications from Event Notifications service, make sure to update the client application with latest IBM Event Notifications client SDK. The customers must update the client application to the latest version in order to make sure that a valid push token is available in the Event Notifications service. Any device still connecting to the old Push Notifications service instance, must be migrated again separately.
  
  If you are sending notifications from both IBM Event Notifications and IBM Push Notifications service, that might result in duplicate messages at the client side. 

