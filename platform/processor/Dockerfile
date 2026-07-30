FROM python:3.12-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends libpq5 gcc libpq-dev libsnappy-dev && \
    rm -rf /var/lib/apt/lists/*
COPY platform/processor/requirements.txt /app/requirements.txt
RUN pip install --no-cache-dir -r /app/requirements.txt
COPY platform/processor /app/processor
COPY platform/contracts/python /app/contracts/python
COPY platform/cache_sync /app/cache_sync
COPY platform/analytics /app/analytics
ENV PYTHONUNBUFFERED=1
WORKDIR /app/processor
CMD ["python", "consume.py"]
