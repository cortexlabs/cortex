from scipy.io.wavfile import read
import numpy as np
import io
import csv


class TensorFlowPredictor:
    def __init__(self, tensorflow_client, config):
        self.client = tensorflow_client
        self.class_names = self.class_names_from_csv("class_names.csv")

    def class_names_from_csv(self, csv_file):
        class_names = []
        with open(csv_file, "r", newline="") as f:
            for row in csv.reader(f, delimiter=","):
                class_names.append(row[2])
        return class_names

    def predict(self, payload):
        rate, data = read(io.BytesIO(payload))
        assert rate == 16000

        result = self.client.predict({"waveform": np.array(data, dtype=np.float32)})
        scores = np.array(result["output_0"]).reshape((-1, 521))

        predicted_class = self.class_names[scores.mean(axis=0).argmax() + 1]
        return predicted_class
