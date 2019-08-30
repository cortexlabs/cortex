from encoder import get_encoder

encoder = get_encoder()


def pre_inference(sample, metadata):
    context = encoder.encode(sample["text"])
    return {"context": [context]}


def post_inference(prediction, metadata):
    response = prediction["response"]["sample"]
    return {encoder.decode(response)}
