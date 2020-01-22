## Upload the Model to S3

First, download the license plate model. Its name is `license_plate.h5`. Download it from [here](https://www.dropbox.com/sh/4ltffycnzfeul01/AACe85GoIzlmjEnIhuh5JQPma?dl=0).

Then, make sure `aws` is configured and upload it to the cortex cluster's bucket.
```bash
aws s3 cp file.txt s3://my-bucket/ --grants read=uri=http://acs.amazonaws.com/groups/global/AllUsers full=emailaddress=user@example.com
```