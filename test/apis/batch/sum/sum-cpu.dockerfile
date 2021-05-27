FROM python:3.8-slim

# Allow statements and log messages to immediately appear in the logs
ENV PYTHONUNBUFFERED True

# Install production dependencies
RUN pip install --no-cache-dir \
    "uvicorn[standard]" \
    gunicorn \
    fastapi \
    pydantic \
    boto3==1.17.72

# Copy local code to the container image.
COPY ./main.py /app/
WORKDIR /app/

ENV PYTHONPATH=/app
ENV CORTEX_PORT=9000

# Run the web service on container startup.
CMD gunicorn -k uvicorn.workers.UvicornWorker --workers 1 --threads 1 --bind :$CORTEX_PORT main:app
