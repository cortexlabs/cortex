import os

from fastapi import FastAPI, status
from fastapi.responses import PlainTextResponse
from pydantic import BaseModel
from transformers import GPT2Tokenizer, GPT2LMHeadModel

app = FastAPI()
device = os.getenv("TARGET_DEVICE", "cpu")
state = {
    "ready": False,
    "tokenizer": None,
    "model": None,
}


@app.on_event("startup")
def startup():
    global state
    state["tokenizer"] = GPT2Tokenizer.from_pretrained("gpt2")
    state["model"] = GPT2LMHeadModel.from_pretrained("gpt2").to(device)
    state["ready"] = True


@app.get("/healthz")
def healthz():
    if state["ready"]:
        return PlainTextResponse("ok")
    return PlainTextResponse("service unavailable", status_code=status.HTTP_503_SERVICE_UNAVAILABLE)


class Request(BaseModel):
    text: str


@app.post("/")
def text_generator(request: Request):
    input_length = len(request.text.split())
    tokens = state["tokenizer"].encode(request.text, return_tensors="pt").to(device)
    prediction = state["model"].generate(tokens, max_length=input_length + 20, do_sample=True)
    return {"text": state["tokenizer"].decode(prediction[0])}
