# WARNING: you are on the master branch, please refer to the examples on the branch that matches your `cortex version`

import numpy as np
import cv2
from .colors import get_color


class BoundBox:
    def __init__(self, xmin, ymin, xmax, ymax, c=None, classes=None):
        self.xmin = xmin
        self.ymin = ymin
        self.xmax = xmax
        self.ymax = ymax

        self.c = c
        self.classes = classes

        self.label = -1
        self.score = -1

    def get_label(self):
        if self.label == -1:
            self.label = np.argmax(self.classes)

        return self.label

    def get_score(self):
        if self.score == -1:
            self.score = self.classes[self.get_label()]

        return self.score


def draw_boxes(image, boxes, overlay_text, labels, obj_thresh, quiet=True):
    for box, overlay in zip(boxes, overlay_text):
        label_str = ""
        label = -1

        for i in range(len(labels)):
            if box.classes[i] > obj_thresh:
                if label_str != "":
                    label_str += ", "
                label_str += labels[i] + " " + str(round(box.get_score() * 100, 2)) + "%"
                label = i
            if not quiet:
                print(label_str)

        if label >= 0:
            if len(overlay) > 0:
                text = label_str + ": [" + " ".join(overlay) + "]"
            else:
                text = label_str
            text = text.upper()
            text_size = cv2.getTextSize(text, cv2.FONT_HERSHEY_SIMPLEX, 1.1e-3 * image.shape[0], 5)
            width, height = text_size[0][0], text_size[0][1]
            region = np.array(
                [
                    [box.xmin - 3, box.ymin],
                    [box.xmin - 3, box.ymin - height - 26],
                    [box.xmin + width + 13, box.ymin - height - 26],
                    [box.xmin + width + 13, box.ymin],
                ],
                dtype="int32",
            )

            # cv2.rectangle(img=image, pt1=(box.xmin,box.ymin), pt2=(box.xmax,box.ymax), color=get_color(label), thickness=5)
            rec = (box.xmin, box.ymin, box.xmax - box.xmin, box.ymax - box.ymin)
            rec = tuple(int(i) for i in rec)
            cv2.rectangle(img=image, rec=rec, color=get_color(label), thickness=3)
            cv2.fillPoly(img=image, pts=[region], color=get_color(label))
            cv2.putText(
                img=image,
                text=text,
                org=(box.xmin + 13, box.ymin - 13),
                fontFace=cv2.FONT_HERSHEY_SIMPLEX,
                fontScale=1e-3 * image.shape[0],
                color=(0, 0, 0),
                thickness=1,
            )

    return image
