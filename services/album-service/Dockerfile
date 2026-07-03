FROM python:3.12-slim
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 PIP_NO_CACHE_DIR=1
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends libpq-dev gcc && rm -rf /var/lib/apt/lists/*
COPY services/album-service/requirements.txt .
COPY shared/ /app/shared/
RUN pip install --upgrade pip && pip install -r requirements.txt
COPY services/album-service/ .
EXPOSE 8002
CMD ["gunicorn", "config.wsgi:application", "--bind", "0.0.0.0:8002", "--workers", "2", "--timeout", "120"]
