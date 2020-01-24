## Upload the Model to S3

First, download the license plate model. Its name is `license_plate.h5`. Download it from [here](https://www.dropbox.com/sh/4ltffycnzfeul01/AACe85GoIzlmjEnIhuh5JQPma?dl=0).

Then, make sure `aws` is configured and upload it to the cortex cluster's bucket.
```bash
aws s3 cp mymodel.h5 s3://my-bucket/path/to/file.h5
```

## To Save an Image as String

```python
# encode image in b64 format
img = open("image.jpg", "rb").read()
img_enc = base64.b64encode().decode("utf-8")

# decode b64 image back to its original form
img_dec = base64.b64decode(img_enc.encode("utf-8"))
jpg_as_np = np.frombuffer(img_dec, dtype=np.uint8)
img_final = cv2.imdecode(jpg_as_np, flags=1)
```