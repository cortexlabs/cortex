import click
import cv2
import requests
import numpy as np

def get_url_image(url_image):
    resp = requests.get(url_image, stream=True).raw
    image = np.asarray(bytearray(resp.read()), dtype="uint8")
    image = cv2.imdecode(image, cv2.IMREAD_COLOR)
    return image

@click.command(help=(
    "Identify license plates in a given image"
    " while outsourcing the predictions using the REST API endpoints."
    " Both API endpoints have to be exported as environment variables."
))
@click.argument("img_url_src", type=str)
@click.argument('yolov3_endpoint', envvar='YOLOV3_ENDPOINT')
@click.argument("crnn_endpoint", envvar="CRNN_ENDPOINT")
def main(yolov3_endpoint, crnn_endpoint, img_url_src):
    image = get_url_image(img_url_src)

if __name__ == "__main__":
    main()