import click, cv2, requests, pickle, base64, json
import numpy as np
import utils.bbox as bbox_utils
import utils.preprocess as preprocess_utils


@click.command(
    help=(
        "Identify license plates in a given image"
        " while outsourcing the predictions using the REST API endpoints."
        " Both API endpoints have to be exported as environment variables."
    )
)
@click.argument("img_url_src", type=str)
@click.argument("yolov3_endpoint", envvar="YOLOV3_ENDPOINT")
@click.argument("crnn_endpoint", envvar="CRNN_ENDPOINT")
@click.option(
    "--output",
    "-o",
    type=str,
    default="prediction.jpg",
    show_default=True,
    help="File to save the prediction to.",
)
def main(img_url_src, yolov3_endpoint, crnn_endpoint, output):

    # get the image in bytes representation
    image = preprocess_utils.get_url_image(img_url_src)
    image_bytes = preprocess_utils.image_to_jpeg_bytes(image)

    # encode image
    image_enc = base64.b64encode(image_bytes).decode("utf-8")
    image_dump = json.dumps({"img": image_enc})

    # make yolov3 api request
    resp = requests.post(
        yolov3_endpoint, data=image_dump, headers={"content-type": "application/json"}
    )

    # parse response
    boxes_raw = resp.json()["boxes"]
    boxes = []
    for b in boxes_raw:
        box = bbox_utils.BoundBox(*b)
        boxes.append(box)

    # purge bounding boxes with a low confidence score
    confidence_score = 0.8
    aux = []
    for b in boxes:
        label = -1
        for i in range(len(b.classes)):
            if b.classes[i] > confidence_score:
                label = i
        if label >= 0:
            aux.append(b)
    boxes = aux
    del aux

    dec_words = []
    if len(boxes) > 0:
        # create set of images of the detected license plates
        lps = []
        for b in boxes:
            lp = image[b.ymin : b.ymax, b.xmin : b.xmax]
            jpeg = preprocess_utils.image_to_jpeg_nparray(lp)
            lps.append(jpeg)

        # encode the cropped license plates
        lps = pickle.dumps(lps, protocol=0)
        lps_enc = base64.b64encode(lps).decode("utf-8")
        lps_dump = json.dumps({"imgs": lps_enc})

        # make crnn api request
        resp = requests.post(
            crnn_endpoint, data=lps_dump, headers={"content-type": "application/json"}
        )

        # parse the response
        dec_lps = resp.json()["license-plates"]
        dec_lps = preprocess_utils.reorder_recognized_words(dec_lps)
        for dec_lp in dec_lps:
            dec_words.append([word[0] for word in dec_lp])

    if len(dec_words) == 0:
        dec_words = [[] for i in range(len(boxes))]

    # draw predictions as overlays on the source image
    draw_image = bbox_utils.draw_boxes(
        image, boxes, overlay_text=dec_words, labels=["LP"], obj_thresh=confidence_score
    )

    # and save it to disk
    cv2.imwrite(output, draw_image)


if __name__ == "__main__":
    main()
