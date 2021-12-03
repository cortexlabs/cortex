FROM python:3.8-slim-buster

ENV PYTHONUNBUFFERED TRUE

COPY app/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt

COPY app /app
WORKDIR /app/
ENV PYTHONPATH=/app

ENV CORTEX_PORT=8080

CMD [ "python3", "-m" , "flask", "run", "--host=0.0.0.0", "--port=8080"]
