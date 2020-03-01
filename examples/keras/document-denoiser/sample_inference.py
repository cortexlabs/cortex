# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import click, cv2, requests, pickle, base64, json
import numpy as np


def image_to_png_nparray(image):
    """
    Convert numpy image to png numpy vector.
    """
    is_success, im_buf_arr = cv2.imencode(".png", image)
    return im_buf_arr


def image_to_png_bytes(image):
    """
    Convert numpy image to bytes-encoded png image.
    """
    buf = image_to_png_nparray(image)
    byte_im = buf.tobytes()
    return byte_im


def get_url_image(url_image):
    """
    Get numpy image from URL image.
    """
    resp = requests.get(url_image, stream=True).raw
    image = np.asarray(bytearray(resp.read()), dtype="uint8")
    image = cv2.imdecode(image, cv2.IMREAD_GRAYSCALE)
    return image


def image_from_bytes(byte_im):
    nparr = np.frombuffer(byte_im, np.uint8)
    img_np = cv2.imdecode(nparr, cv2.IMREAD_GRAYSCALE)
    return img_np


@click.command(help=("Clean dirty documents whilst using" " the given REST API endpoint."))
@click.argument("img_url_src", type=str)
@click.argument("endpoint", envvar="ENDPOINT")
@click.option(
    "--output",
    "-o",
    type=str,
    default="prediction.png",
    show_default=True,
    help="File to save the prediction to.",
)
def main(img_url_src, endpoint, output):

    # get the image in bytes representation
    image = get_url_image(img_url_src)
    image_bytes = image_to_png_bytes(image)

    # encode image
    image_enc = base64.b64encode(image_bytes).decode("utf-8")
    image_dump = json.dumps({"img": image_enc})

    # make yolov3 api request
    resp = requests.post(endpoint, data=image_dump, headers={"content-type": "application/json"})

    # parse the response
    img_enc = resp.json()
    img_enc = json.loads(img_enc)["img_pred"]
    image_bytes = base64.b64decode(img_enc.encode("utf-8"))
    img_as_np = image_from_bytes(image_bytes)

    # write the prediction
    cv2.imwrite(output, img_as_np)


if __name__ == "__main__":
    main()
