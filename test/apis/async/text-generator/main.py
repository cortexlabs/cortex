import os

from fastapi import FastAPI, Response, status
from pydantic import BaseModel
from transformers import GPT2Tokenizer, GPT2LMHeadModel


class Request(BaseModel):
    text: str


state = {
    "model_ready": False,
    "tokenizer": None,
    "model": None,
}
device = os.getenv("TARGET_DEVICE", "cpu")
app = FastAPI()


@app.on_event("startup")
def startup():
    global state
    state["tokenizer"] = GPT2Tokenizer.from_pretrained("gpt2")
    state["model"] = GPT2LMHeadModel.from_pretrained("gpt2").to(device)
    state["model_ready"] = True


@app.get("/healthz")
def healthz(response: Response):
    if not state["model_ready"]:
        response.status_code = status.HTTP_503_SERVICE_UNAVAILABLE


@app.post("/")
def text_generator(request: Request):
    input_length = len(request.text.split())
    tokens = state["tokenizer"].encode(request.text, return_tensors="pt").to(device)
    prediction = state["model"].generate(tokens, max_length=input_length + 20, do_sample=True)
    return {
        "prediction": state["tokenizer"].decode(prediction[0]),
    }
