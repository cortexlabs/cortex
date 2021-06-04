FROM python:3.7-slim

ENV PYTHONUNBUFFERED TRUE

COPY app/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

COPY app /app
WORKDIR /app/

CMD exec python app/main.py
