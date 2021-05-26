FROM nvidia/cuda:10.2-cudnn8-runtime-ubuntu18.04

RUN apt-get update \
    && apt-get install \
        python3 \
        python3-pip \
        pkg-config \
        git \
        build-essential \
        cmake -y \
    && apt-get clean -qq && rm -rf /var/lib/apt/lists/*

# Allow statements and log messages to immediately appear in the logs
ENV PYTHONUNBUFFERED True

# Install production dependencies
RUN pip3 install --no-cache-dir "uvicorn[standard]" gunicorn fastapi pydantic transformers==3.0.* torch==1.7.*

# Copy local code to the container image.
COPY . /app
WORKDIR /app/

ENV PYTHONPATH=/app
ENV CORTEX_PORT=9000

# Run the web service on container startup.
CMD gunicorn -k uvicorn.workers.UvicornWorker --workers 1 --threads 1 --bind :$CORTEX_PORT main:app
