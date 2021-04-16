# Alerts

Cortex supports setting alerts for your APIs out-of-the-box. Alerts are an effective way of identifying problems in your system moments after they occur. This allows you to take action as soon as they happen to minimize disruptions to your APIs.

The following dashboards can be configured with alerts:

- RealtimeAPI
- BatchAPI
- Cluster resources
- Node resources

On this page, we're going through the process of configuring alerts for a realtime API. The same principles apply to the others as well. Alerts can be configured for a variety of notification channels such as for Slack, Discord, Microsoft Teams, PagerDuty, Telegram and traditional webhooks, but on this page, we'll only be focusing on integrating Slack.

The following steps assume that you know how to create a Slack channel and know how to access the Grafana dashboard of an API. If you don't know how to access the Grafana dashboard of your API, make sure you check out [this page](metrics.md) first.

## Slack channel

Create a slack channel on your team's Slack. We'll give ours the `cortex-alerts` name.

Add an _Incoming Webhook_ app to your channel and retrieve the generated webhook URL. It will look like something like `https://hooks.slack.com/services/<XXX>/<YYY>/<ZZZ>`.

## Notification channel

Go to your Grafana and on the left-hand side panel, hover over the alerting bell and select _"Notification channels"_.

![notification channel](https://user-images.githubusercontent.com/26958764/114937638-b6667780-9e46-11eb-963a-8a53e5655c3d.png)

## Create channel

Click on _"Create channel"_ and add the name of your channel, select _Slack_ as the type of the channel and copy-paste the webhook URL generated in the earlier steps.

![create channel](https://user-images.githubusercontent.com/26958764/114937856-06ddd500-9e47-11eb-8f47-47b043b0bb5c.png)

Afterwards, hit _"Test"_ and see if a sample notification is sent to your Slack channel. If the test is positive, click on _"Save"_ next.

![test notification](https://user-images.githubusercontent.com/26958764/114938358-b2872500-9e47-11eb-87aa-ee818aae4cd0.png)

## Create alerts

With a functional notification channel, we can now go and create alerts for our APIs/cluster. For all of our examples, we are using the `mpg-estimator` API as a sandbox API.

![sandbox api](https://user-images.githubusercontent.com/26958764/114939831-a8662600-9e49-11eb-8774-fbac3ce627d9.png)

### API replica threshold alert

Let's create an alert for the _"Active Replicas"_ panel. We want to send notifications every time when the number of replicas for the given API exceeds a certain threshold.

Edit the _"Active Replicas"_ panel.

![edit active replicas](https://user-images.githubusercontent.com/26958764/114941416-d2b8e300-9e4b-11eb-8def-ee64535fc799.png)

Create a copy of the primary query by clicking on the duplicate icon.

![copy primary query](https://user-images.githubusercontent.com/26958764/114941557-fe3bcd80-9e4b-11eb-8f69-b9d43ff8eb28.png)

In the copied query, replace the `$api_name` variable with the name of the API you want to create the alert for. In our case, it's the `mpg-estimator`. Also, click on the eye icon to disable the query from being shown on the graph - otherwise, you'll see duplicates.

![remove duplicates](https://user-images.githubusercontent.com/26958764/114941701-275c5e00-9e4c-11eb-991b-34d660d0d05c.png)

Go to the _"Alert"_ tab and press on _"Create Alert"_.

![alert tab](https://user-images.githubusercontent.com/26958764/114941779-40fda580-9e4c-11eb-9aee-514e6b4832ba.png)

Configure your alert like in the following example and hit _"Apply"_.

![alert config](https://user-images.githubusercontent.com/26958764/114944749-d7cc6100-9e50-11eb-9b6b-b2c3dabcc78c.png)

The next time the threshold is hit, that will send a notification to your Slack channel.

![slack notification](https://user-images.githubusercontent.com/26958764/114948423-a3a86e80-9e57-11eb-8717-94e456a15298.png)

### In-flight requests spike alert

Let's take the _"In-Flight Requests"_ panel. We want to send an alert if the metric goes over 50 in-flight requests. For this, follow the same set of instructions as for the previous alert, but this time configure the alert to be like in the following screenshot:

![instant in-flight requests](https://user-images.githubusercontent.com/26958764/114949182-1bc36400-9e59-11eb-9c19-0d788872a388.png)

An alert coming for this will look like this:

![slack notification](https://user-images.githubusercontent.com/26958764/114949593-000c8d80-9e5a-11eb-8cb5-b2c9a2b344e8.png)

#### Memory usage alert

Let's do one more for the _"Avg Memory Usage"_ panel. We want to send an alert if the average memory usage per API replica exceeds its memory request. For this, we need to follow the same set of instructions as for the first alert, but this time the hidden query needs to be expressed as the ratio between the memory usage and memory request.

![memory usage query](https://user-images.githubusercontent.com/26958764/114951903-f1c07080-9e5d-11eb-9aaf-898d46efb7ef.png)

The memory usage alert needs to be defined like in the following screenshot:

![memory usage alert](https://user-images.githubusercontent.com/26958764/114951782-bfaf0e80-9e5d-11eb-834d-e48ab3546d3c.png)

The alert that will get sent to the Slack channel will be the following:

![slack notification](https://user-images.githubusercontent.com/26958764/114952346-bd00e900-9e5e-11eb-879a-5851dab7630b.png)

## Persistent changes

To save your changes permanently, go back to your dashboard and click on the floppy-disk button on the top-right corner.

![floppy-disk icon](https://user-images.githubusercontent.com/26958764/114953264-af4c6300-9e60-11eb-8095-40e438c125d8.png)

Copy-paste the JSON to clipboard.

![copy to clipboard](https://user-images.githubusercontent.com/26958764/114953338-d6a33000-9e60-11eb-8390-0f24704c5b7d.png)

Click on the settings button on the top-right corner of your dashboard.

![settings button](https://user-images.githubusercontent.com/26958764/114953437-00f4ed80-9e61-11eb-91f6-4b669ffe0c16.png)

Go to the _"JSON Model"_ section and replace the JSON model with the one you've copy-pasted to your clipboard. Then hit _"Save Changes"_.

![json model](https://user-images.githubusercontent.com/26958764/114953473-1ec25280-9e61-11eb-8fcc-12615b73067a.png)

Your dashboard now has stored the alert configuration permanently.

## Multiple APIs alerts

Due to how Grafana was built, you'll need to re-do the steps of setting a given alert for different APIs. That's because alerts on Grafana don't currently support template or transformation queries.
