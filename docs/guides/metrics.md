# View API metrics

_WARNING: you are on the master branch, please refer to the docs on the branch that matches your `cortex version`_

The `cortex get` and `cortex get API_NAME` commands display the request time (averaged over the past 2 weeks) and response code counts (summed over the past 2 weeks) for your API(s):

```text
$ cx get

env   api                         status   up-to-date   requested   last update   avg request   2XX
aws   iris-classifier             live     1            1           17m           24ms          1223
aws   text-generator              live     1            1           8m            180ms         433
aws   image-classifier-resnet50   live     2            2           1h            32ms          1121126
```

The `cortex get API_NAME` command also provides a link to a CloudWatch Metrics dashboard containing this information:

![dashboard](https://user-images.githubusercontent.com/808475/86186297-8cc5a500-baed-11ea-885f-d5c301b049eb.png)

**responses per minute**

Shows the number of 2XX, 4XX, and 5XX responses per minute.

**median response time**

Shows the median response time for requests, over 1-minute periods (measured in milliseconds).

**p99 response time**

Shows the p99 response time for requests, over 1-minute periods (measured in milliseconds).

**total in-flight requests**

Shows the total number of in-flight requests.

The [note](#note-regarding-metric-intervals) below applies to this plot.

**avg in-flight requests per replica**

Shows the average number of in-flight requests per replica.

The [note](#note-regarding-metric-intervals) below applies to this plot.

**active replicas**

Shows the number of active replicas.

The [note](#note-regarding-metric-intervals) below applies to this plot.

---

#### note regarding metric intervals

The referenced widget is aggregated over 10 second intervals because each replica reports its in-flight requests once per 10 seconds. This plot is only available for the last 3 hours (because second-granular data is aggregated to minute-granular data after 3 hours). To plot data older than 3 hours, instead change the period to 1 minute, and divide the y-axis by 6 to (since the metrics are reported every 10 seconds).*
