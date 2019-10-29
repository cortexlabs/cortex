from encoder import get_encoder

encoder = get_encoder()


def pre_inference(sample, signature, metadata):
    context = encoder.encode(sample["text"])
    return {"context": [context]}


def post_inference(prediction, signature, metadata):
    response = prediction["sample"]
    return encoder.decode(response)
