# this is an example for cortex release 0.14 and may not deploy correctly on other releases of cortex

import click, cv2, requests, pickle, base64, json
import numpy as np
from utils.bbox import BoundBox, draw_boxes
from statistics import mean


def image_to_jpeg_nparray(image, quality=[int(cv2.IMWRITE_JPEG_QUALITY), 95]):
    """
    Convert numpy image to jpeg numpy vector.
    """
    is_success, im_buf_arr = cv2.imencode(".jpg", image, quality)
    return im_buf_arr


def image_to_jpeg_bytes(image, quality=[int(cv2.IMWRITE_JPEG_QUALITY), 95]):
    """
    Convert numpy image to bytes-encoded jpeg image.
    """
    buf = image_to_jpeg_nparray(image, quality)
    byte_im = buf.tobytes()
    return byte_im


def get_url_image(url_image):
    """
    Get numpy image from URL image.
    """
    resp = requests.get(url_image, stream=True).raw
    image = np.asarray(bytearray(resp.read()), dtype="uint8")
    image = cv2.imdecode(image, cv2.IMREAD_COLOR)
    return image


def reorder_recognized_words(detected_images):
    """
    Reorder the detected words in each image based on the average horizontal position of each word.
    Sorting them in ascending order.
    """

    reordered_images = []
    for detected_image in detected_images:

        # computing the mean average position for each word
        mean_horizontal_positions = []
        for words in detected_image:
            box = words[1]
            y_positions = [point[0] for point in box]
            mean_y_position = mean(y_positions)
            mean_horizontal_positions.append(mean_y_position)
        indexes = np.argsort(mean_horizontal_positions)

        # and reordering them
        reordered = []
        for index, words in zip(indexes, detected_image):
            reordered.append(detected_image[index])
        reordered_images.append(reordered)

    return reordered_images


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
    image = get_url_image(img_url_src)
    image_bytes = image_to_jpeg_bytes(image)

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
        box = BoundBox(*b)
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
            jpeg = image_to_jpeg_nparray(lp)
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
        dec_lps = reorder_recognized_words(dec_lps)
        for dec_lp in dec_lps:
            dec_words.append([word[0] for word in dec_lp])

    if len(dec_words) == 0:
        dec_words = [[] for i in range(len(boxes))]

    # draw predictions as overlays on the source image
    draw_image = draw_boxes(
        image, boxes, overlay_text=dec_words, labels=["LP"], obj_thresh=confidence_score
    )

    # and save it to disk
    cv2.imwrite(output, draw_image)


if __name__ == "__main__":
    main()
