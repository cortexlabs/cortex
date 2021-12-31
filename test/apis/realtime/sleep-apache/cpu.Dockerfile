FROM python:3.8-slim

ENV PYTHONUNBUFFERED TRUE

COPY app/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

COPY app /app
WORKDIR /app/
ENV PYTHONPATH=/app

CMD uvicorn --workers 1 --limit-concurrency 2 --host 0.0.0.0 --port 9000 main:app
