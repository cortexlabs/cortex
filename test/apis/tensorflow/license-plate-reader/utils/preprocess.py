import numpy as np
import cv2, requests
from statistics import mean


def get_url_image(url_image):
    """
    Get numpy image from URL image.
    """
    resp = requests.get(url_image, stream=True).raw
    image = np.asarray(bytearray(resp.read()), dtype="uint8")
    image = cv2.imdecode(image, cv2.IMREAD_COLOR)
    return image


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
