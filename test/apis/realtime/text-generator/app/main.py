import os

from fastapi import FastAPI, status
from fastapi.responses import PlainTextResponse
from pydantic import BaseModel
from transformers import GPT2Tokenizer, GPT2LMHeadModel

app = FastAPI()
app.device = os.getenv("TARGET_DEVICE", "cpu")
app.ready = False


@app.on_event("startup")
def startup():
    app.tokenizer = GPT2Tokenizer.from_pretrained("gpt2")
    app.model = GPT2LMHeadModel.from_pretrained("gpt2").to(app.device)
    app.ready = True


@app.get("/healthz")
def healthz():
    if app.ready:
        return PlainTextResponse("ok")
    return PlainTextResponse("service unavailable", status_code=status.HTTP_503_SERVICE_UNAVAILABLE)


class Body(BaseModel):
    text: str


@app.post("/")
def text_generator(body: Body):
    input_length = len(body.text.split())
    tokens = app.tokenizer.encode(body.text, return_tensors="pt").to(app.device)
    prediction = app.model.generate(tokens, max_length=input_length + 20, do_sample=True)
    return {"text": app.tokenizer.decode(prediction[0])}
