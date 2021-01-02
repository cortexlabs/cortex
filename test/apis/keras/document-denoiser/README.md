# Clean Dirty Documents w/ Autoencoders

This example model cleans text documents of anything that isn't text (aka noise): coffee stains, old wear artifacts, etc. You can inspect the notebook that has been used to train the model [here](trainer.ipynb).

Here's a collage of input texts and predictions.

![Imgur](https://i.imgur.com/M4Mjz2l.jpg)

*Figure 1 - The dirty documents are on the left side and the cleaned ones are on the right*

## Sample Prediction

Once this model is deployed, get the API endpoint by running `cortex get document-denoiser`.

Now let's take a sample image like this one.

![Imgur](https://i.imgur.com/JJLfFxB.png)

Export the endpoint & the image's URL by running
```bash
export ENDPOINT=<API endpoint>
export IMAGE_URL=https://i.imgur.com/JJLfFxB.png
```

Then run the following piped commands
```bash
curl "${ENDPOINT}" -X POST -H "Content-Type: application/json" -d '{"url":"'${IMAGE_URL}'"}' |
sed 's/"//g' |
base64 -d > prediction.png
```

Once this has run, we'll see a `prediction.png` file saved to the disk. This is the result.

![Imgur](https://i.imgur.com/PRB2oS8.png)

As it can be seen, the text document has been cleaned of any noise. Success!

---

Here's a short list of URLs of other text documents in image format that can be cleaned using this model. Export these links to `IMAGE_URL` variable:

* https://i.imgur.com/6COQ46f.png
* https://i.imgur.com/alLI83b.png
* https://i.imgur.com/QVoSTuu.png
