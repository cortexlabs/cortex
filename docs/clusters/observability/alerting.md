# Alerting

Cortex supports setting alerts for your APIs out-of-the-box. Alerts are an effective way of identifying problems in your system as they occur.

The following dashboards can be configured with alerts:

- RealtimeAPI
- BatchAPI
- Cluster resources
- Node resources

This page demonstrates the process for configuring alerts for a realtime API. The same principles apply to the others as well. Alerts can be configured for a variety of notification channels such as for Slack, Discord, Microsoft Teams, PagerDuty, Telegram and traditional webhooks. In this example, we'll use Slack.

If you don't know how to access the Grafana dashboard for your API, make sure you check out [this page](metrics.md) first.

## Create a Slack channel

Create a slack channel on your team's Slack workspace. We'll name ours "cortex-alerts".

Add an _Incoming Webhook_ to your channel and retrieve the webhook URL. It will look like something like `https://hooks.slack.com/services/<XXX>/<YYY>/<ZZZ>`.

## Create a Grafana notification channel

Go to Grafana and on the left-hand side panel, hover over the alerting bell and select _"Notification channels"_.

![](https://user-images.githubusercontent.com/26958764/114937638-b6667780-9e46-11eb-963a-8a53e5655c3d.png)

Click on _"Create channel"_, add the name of your channel, select _Slack_ as the type of the channel, and paste your Slack webhook URL.

![](https://user-images.githubusercontent.com/26958764/114937856-06ddd500-9e47-11eb-8f47-47b043b0bb5c.png)

Click _"Test"_ and see if a sample notification is sent to your Slack channel. If the message goes through, click on _"Save"_.

![](https://user-images.githubusercontent.com/26958764/114938358-b2872500-9e47-11eb-87aa-ee818aae4cd0.png)

## Create alerts

Now that the notification channel is functioning properly, we can create alerts for our APIs and cluster. For all of our examples, we are using the `mpg-estimator` API as an example.

![](https://user-images.githubusercontent.com/26958764/114939831-a8662600-9e49-11eb-8774-fbac3ce627d9.png)

### API replica threshold alert

Let's create an alert for the _"Active Replicas"_ panel. We want to send notifications every time the number of replicas for the given API exceeds a certain threshold.

Edit the _"Active Replicas"_ panel.

![](https://user-images.githubusercontent.com/26958764/114941416-d2b8e300-9e4b-11eb-8def-ee64535fc799.png)

Create a copy of the primary query by clicking on the "duplicate" icon.

![](https://user-images.githubusercontent.com/26958764/114941557-fe3bcd80-9e4b-11eb-8f69-b9d43ff8eb28.png)

In the copied query, replace the `$api_name` variable with the name of the API you want to create the alert for. In our case, it's `mpg-estimator`. Also, click on the eye icon to disable the query from being shown on the graph - otherwise, you'll see duplicates.

![](https://user-images.githubusercontent.com/26958764/114941701-275c5e00-9e4c-11eb-991b-34d660d0d05c.png)

Go to the _"Alert"_ tab and click _"Create Alert"_.

![](https://user-images.githubusercontent.com/26958764/114941779-40fda580-9e4c-11eb-9aee-514e6b4832ba.png)

Configure your alert like in the following example and click _"Apply"_.

![](https://user-images.githubusercontent.com/26958764/114944749-d7cc6100-9e50-11eb-9b6b-b2c3dabcc78c.png)

The next time the threshold is exceeded, a notification will be sent to your Slack channel.

![](https://user-images.githubusercontent.com/26958764/114948423-a3a86e80-9e57-11eb-8717-94e456a15298.png)

### In-flight requests spike alert

Let's add an alert on the _"In-Flight Requests"_ panel. We want to send an alert if the metric exceeds 50 in-flight requests. For this, follow the same set of instructions as for the previous alert, but this time configure the alert to match the following screenshot:

![](https://user-images.githubusercontent.com/26958764/114949182-1bc36400-9e59-11eb-9c19-0d788872a388.png)

An alert triggered for this will look like:

![](https://user-images.githubusercontent.com/26958764/114949593-000c8d80-9e5a-11eb-8cb5-b2c9a2b344e8.png)

#### Memory usage alert

Let's add another alert, this time for the _"Avg Memory Usage"_ panel. We want to send an alert if the average memory usage per API replica exceeds its memory request. For this, we need to follow the same set of instructions as for the first alert, but this time the hidden query needs to be expressed as the ratio between the memory usage and memory request:

![](https://user-images.githubusercontent.com/26958764/114951903-f1c07080-9e5d-11eb-9aaf-898d46efb7ef.png)

The memory usage alert can to be defined like in the following screenshot:

![](https://user-images.githubusercontent.com/26958764/114951782-bfaf0e80-9e5d-11eb-834d-e48ab3546d3c.png)

The resulting alert will look like this:

![](https://user-images.githubusercontent.com/26958764/114952346-bd00e900-9e5e-11eb-879a-5851dab7630b.png)

## Persistent changes

To save your changes permanently, go back to your dashboard and click on the save icon on the top-right corner.

![](https://user-images.githubusercontent.com/26958764/114953264-af4c6300-9e60-11eb-8095-40e438c125d8.png)

Copy the JSON to your clipboard.

![](https://user-images.githubusercontent.com/26958764/114953338-d6a33000-9e60-11eb-8390-0f24704c5b7d.png)

Click on the settings button on the top-right corner of your dashboard.

![](https://user-images.githubusercontent.com/26958764/114953437-00f4ed80-9e61-11eb-91f6-4b669ffe0c16.png)

Go to the _"JSON Model"_ section and replace the JSON with the one you've copied to your clipboard. Then click _"Save Changes"_.

![](https://user-images.githubusercontent.com/26958764/114953473-1ec25280-9e61-11eb-8fcc-12615b73067a.png)

Your dashboard now has stored the alert configuration permanently.

## Multiple APIs alerts

Due to how Grafana was built, you'll need to re-do the steps of setting a given alert for each individual API. That's because Grafana doesn't currently support alerts on template or transformation queries.

## Enabling email alerts

It is possible to manually configure SMTP to enable email alerts (we plan on automating this process, see [#2210](https://github.com/cortexlabs/cortex/issues/2210)).

**Step 1**

Install [kubectl](../advanced/kubectl.md).

**Step 2**

```bash
kubectl create secret generic grafana-smtp \
    --from-literal=GF_SMTP_ENABLED=true \
    --from-literal=GF_SMTP_HOST=<SMTP-HOST> \
    --from-literal=GF_SMTP_USER=<EMAIL-ADDRESS> \
    --from-literal=GF_SMTP_FROM_ADDRESS=<EMAIL-ADDRESS> \
    --from-literal=GF_SMTP_PASSWORD=<EMAIL-PASSWORD>
```

The `<SMTP-HOST>` varies from provider to provider (e.g. Gmail's is `smtp.gmail.com:587`).

**Step 3**

Edit Grafana's statefulset by running `kubectl edit statefulset grafana` (this will open a code editor). Inside the container named `grafana` (in the `containers` section), add an `envFrom` section that will mount the SMTP secret. Here is an example of what it looks like after the addition:

<!-- CORTEX_VERSION_README -->

```yaml
# ...
containers:
- env:
  - name: GF_SERVER_ROOT_URL
    value: '%(protocol)s://%(domain)s:%(http_port)s/dashboard'
  - name: GF_SERVER_SERVE_FROM_SUB_PATH
    value: "true"
  - name: GF_USERS_DEFAULT_THEME
    value: light
  envFrom:
    - secretRef:
        name: grafana-smtp
  image: quay.io/cortexlabs/grafana:0.42.2
  imagePullPolicy: IfNotPresent
  name: grafana
# ...
```

Save and close your editor.

It will take 30-60 seconds for Grafana to restart, after which you can access the dashboard. You can check the logs with `kubectl logs -f grafana-0`.
