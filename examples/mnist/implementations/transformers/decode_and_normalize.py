import base64
from PIL import Image
from io import BytesIO
import numpy as np
import math


def transform_python(input):
    decoded = base64.b64decode(input)
    decoded_image = np.asarray(Image.open(BytesIO(decoded)), dtype=np.uint8)

    # reimplmenting tf.per_image_standardization
    # https://www.tensorflow.org/api_docs/python/tf/image/per_image_standardization
    adjusted_stddev = max(np.std(decoded_image), 1.0 / math.sqrt(decoded_image.size))
    standardized_image = (decoded_image - np.mean(decoded_image)) / adjusted_stddev
    return standardized_image.flatten().tolist()
