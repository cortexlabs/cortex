import requests
import numpy as np
import cv2


def get_url_image(url_image):
    """
    Get numpy image from URL image.
    """
    resp = requests.get(url_image, stream=True).raw
    image = np.asarray(bytearray(resp.read()), dtype="uint8")
    image = cv2.imdecode(image, cv2.IMREAD_COLOR)
    image = cv2.cvtColor(image, cv2.COLOR_BGR2RGB)
    return image


class Handler:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client

        # for image classifiers
        classes = requests.get(config["image-classifier-classes"]).json()
        self.image_classes = [classes[str(k)][1] for k in range(len(classes))]

        # assign "models"' key value to self.config for ease of use
        self.config = config["models"]

        # for iris classifier
        self.iris_labels = self.config["iris"]["labels"]

    def handle_post(self, payload, query_params):
        model_name = query_params["model"]
        model_version = query_params.get("version", "latest")
        predicted_label = None

        if model_name == "iris":
            prediction = self.client.predict(payload["input"], model_name, model_version)
            predicted_class_id = int(prediction["class_ids"][0])
            predicted_label = self.iris_labels[predicted_class_id]

        elif model_name in ["resnet50", "inception"]:
            predicted_label = self.predict_image_classifier(model_name, payload["url"])

        return {"label": predicted_label, "model": {"model": model_name, "version": model_version}}

    def predict_image_classifier(self, model, img_url):
        img = get_url_image(img_url)
        img = cv2.resize(
            img, tuple(self.config[model]["input_shape"]), interpolation=cv2.INTER_NEAREST
        )
        if model == "inception":
            img = img.astype("float32") / 255
        img = {self.config[model]["input_key"]: img[np.newaxis, ...]}

        results = self.client.predict(img, model)[self.config[model]["output_key"]]
        result = np.argmax(results)
        if model == "inception":
            result -= 1
        predicted_label = self.image_classes[result]

        return predicted_label
