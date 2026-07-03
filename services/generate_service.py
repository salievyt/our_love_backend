#!/usr/bin/env python
"""Generate a standard Django microservice for Our Love project.

Usage: python generate_service.py <service_name> <app_name> <port>
Example: python generate_service.py album album 8002
"""
import os
import sys
from pathlib import Path

SERVICES_DIR = Path(__file__).parent
SHARED_DIR = SERVICES_DIR.parent / "shared"

SERVICE_TEMPLATE = '''import os
from datetime import timedelta
from pathlib import Path

import environ

env = environ.Env(DEBUG=(bool, False))
BASE_DIR = Path(__file__).resolve().parent.parent
env.read_env(os.path.join(BASE_DIR, ".env"))

SECRET_KEY = env("SECRET_KEY", default="change-me-{service}-service")
DEBUG = env("DEBUG")
ALLOWED_HOSTS = env.list("SERVICE_ALLOWED_HOSTS", default=["localhost", "127.0.0.1", "{service}-service"])

DJANGO_APPS = [
    "django.contrib.admin",
    "django.contrib.auth",
    "django.contrib.contenttypes",
    "django.contrib.sessions",
    "django.contrib.messages",
    "django.contrib.staticfiles",
]
THIRD_PARTY_APPS = [
    "rest_framework",
    "corsheaders",
    "drf_spectacular",
    "django_filters",
]
LOCAL_APPS = ["apps.{app}"]
INSTALLED_APPS = DJANGO_APPS + THIRD_PARTY_APPS + LOCAL_APPS

MIDDLEWARE = [
    "corsheaders.middleware.CorsMiddleware",
    "django.middleware.security.SecurityMiddleware",
    "django.contrib.sessions.middleware.SessionMiddleware",
    "django.middleware.common.CommonMiddleware",
    "django.middleware.csrf.CsrfViewMiddleware",
    "django.contrib.auth.middleware.AuthenticationMiddleware",
    "django.contrib.messages.middleware.MessageMiddleware",
    "django.middleware.clickjacking.XFrameOptionsMiddleware",
]

ROOT_URLCONF = "config.urls"
WSGI_APPLICATION = "config.wsgi.application"

DATABASES = {{
    "default": {{
        "ENGINE": "django.db.backends.postgresql",
        "NAME": env("DB_NAME", default="outlove_{service}"),
        "USER": env("DB_USER", default="outlove"),
        "PASSWORD": env("DB_PASSWORD", default="outlove_secret"),
        "HOST": env("DB_HOST", default="{service}-db"),
        "PORT": env("DB_PORT", default="5432"),
    }}
}}

REST_FRAMEWORK = {{
    "DEFAULT_AUTHENTICATION_CLASSES": (
        "shared.auth_backend.MicroserviceJWTAuthentication",
    ),
    "DEFAULT_PERMISSION_CLASSES": (
        "rest_framework.permissions.IsAuthenticated",
    ),
    "DEFAULT_PAGINATION_CLASS": "rest_framework.pagination.PageNumberPagination",
    "PAGE_SIZE": 20,
    "DEFAULT_FILTER_BACKENDS": [
        "django_filters.rest_framework.DjangoFilterBackend",
        "rest_framework.filters.SearchFilter",
        "rest_framework.filters.OrderingFilter",
    ],
    "DEFAULT_SCHEMA_CLASS": "drf_spectacular.openapi.AutoSchema",
}}

SIMPLE_JWT = {{
    "ACCESS_TOKEN_LIFETIME": timedelta(days=1),
    "REFRESH_TOKEN_LIFETIME": timedelta(days=30),
    "AUTH_HEADER_TYPES": ("Bearer",),
    "AUTH_TOKEN_CLASSES": ("rest_framework_simplejwt.tokens.AccessToken",),
}}

LANGUAGE_CODE = "ru"
TIME_ZONE = "Europe/Moscow"
USE_I18N = True
USE_TZ = True
STATIC_URL = "/static/"
STATIC_ROOT = BASE_DIR / "static"
MEDIA_URL = "/media/"
MEDIA_ROOT = BASE_DIR / "media"
DEFAULT_AUTO_FIELD = "django.db.models.BigAutoField"
CORS_ALLOWED_ORIGINS = env.list("CORS_ALLOWED_ORIGINS", default=["http://localhost:3000"])
CORS_ALLOW_CREDENTIALS = True

SPECTACULAR_SETTINGS = {{
    "TITLE": "Our Love — {title} Service",
    "DESCRIPTION": "{title} microservice",
    "VERSION": "1.0.0",
}}
'''


def create_file(path, content):
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content)
    print(f"  Created: {path.relative_to(SERVICES_DIR.parent)}")


def generate_service(service, app, port, title=None):
    if title is None:
        title = service.capitalize()
    
    service_dir = SERVICES_DIR / f"{service}-service"
    app_dir = service_dir / "apps" / app
    config_dir = service_dir / "config"
    
    # config/settings.py
    create_file(
        config_dir / "settings.py",
        SERVICE_TEMPLATE.format(service=service, app=app, title=title)
    )
    
    # config/urls.py
    create_file(
        config_dir / "urls.py",
        f'''from django.contrib import admin
from django.urls import path, include
from drf_spectacular.views import SpectacularAPIView, SpectacularSwaggerView

urlpatterns = [
    path("admin/", admin.site.urls),
    path("api/v1/{service}/", include("apps.{app}.urls")),
    path("api/schema/", SpectacularAPIView.as_view(), name="schema"),
    path("api/docs/", SpectacularSwaggerView.as_view(url_name="schema"), name="swagger-ui"),
]
'''
    )
    
    # config/wsgi.py
    create_file(
        config_dir / "wsgi.py",
        f'''import os
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent.parent.parent.parent / "shared"))
from django.core.wsgi import get_wsgi_application
os.environ.setdefault("DJANGO_SETTINGS_MODULE", "config.settings")
application = get_wsgi_application()
'''
    )
    
    # manage.py
    create_file(
        service_dir / "manage.py",
        f'''#!/usr/bin/env python
import os
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).resolve().parent.parent.parent / "shared"))
os.environ.setdefault("DJANGO_SETTINGS_MODULE", "config.settings")
from django.core.management import execute_from_command_line
execute_from_command_line(sys.argv)
'''
    )
    
    # Dockerfile
    create_file(
        service_dir / "Dockerfile",
        f'''FROM python:3.12-slim
ENV PYTHONDONTWRITEBYTECODE=1 PYTHONUNBUFFERED=1 PIP_NO_CACHE_DIR=1
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends libpq-dev gcc && rm -rf /var/lib/apt/lists/*
COPY services/{service}-service/requirements.txt .
COPY shared/ /app/shared/
RUN pip install --upgrade pip && pip install -r requirements.txt
COPY services/{service}-service/ .
EXPOSE {port}
CMD ["gunicorn", "config.wsgi:application", "--bind", "0.0.0.0:{port}", "--workers", "2", "--timeout", "120"]
'''
    )
    
    # requirements.txt
    create_file(
        service_dir / "requirements.txt",
        '''django>=5.1,<6.0
djangorestframework>=3.15,<4.0
djangorestframework-simplejwt>=5.4,<6.0
django-cors-headers>=4.6,<5.0
django-filter>=25.1,<26.0
drf-spectacular>=0.28,<1.0
django-environ>=0.12,<1.0
gunicorn>=23.0,<24.0
psycopg2-binary>=2.9,<3.0
PyJWT>=2.10,<3.0
'''
    )
    
    # apps/__init__.py
    create_file(app_dir / "__init__.py", "")
    
    # apps/apps.py
    create_file(
        app_dir / "apps.py",
        f'''from django.apps import AppConfig

class {title.replace(" ", "")}Config(AppConfig):
    default_auto_field = "django.db.models.BigAutoField"
    name = "apps.{app}"
    verbose_name = "{title}"
'''
    )
    
    # apps/urls.py
    create_file(
        app_dir / "urls.py",
        f'''from django.urls import path
from apps.{app} import views

urlpatterns = [
    # TODO: Add {app} endpoints
]
'''
    )
    
    # apps/views.py
    create_file(
        app_dir / "views.py",
        f'''from rest_framework import generics
from rest_framework.permissions import IsAuthenticated

# TODO: Add {app} views
'''
    )
    
    # apps/models.py
    create_file(
        app_dir / "models.py",
        f'''from django.db import models

# TODO: Add {app} models
'''
    )
    
    # apps/admin.py
    create_file(
        app_dir / "admin.py",
        f'''from django.contrib import admin

# TODO: Register {app} models
'''
    )
    
    # apps/serializers.py
    create_file(
        app_dir / "serializers.py",
        f'''from rest_framework import serializers

# TODO: Add {app} serializers
'''
    )
    
    print(f"\\n✅ Service '{service}' created on port {port}\\n")


if __name__ == "__main__":
    services = [
        ("album", "albums", 8002, "Album"),
        ("task", "tasks", 8003, "Task"),
        ("question", "questions", 8004, "Question"),
        ("diary", "diary", 8005, "Diary"),
        ("mood", "moods", 8006, "Mood"),
        ("place", "places", 8007, "Place"),
        ("capsule", "capsules", 8008, "Capsule"),
        ("date", "dates", 8009, "Date"),
    ]
    
    print("=== Generating Our Love Microservices ===\\n")
    for service, app, port, title in services:
        generate_service(service, app, port, title)
    
    print("=== All services generated! ===")
    print("Next steps:")
    print("1. Add models, views, serializers to each service")
    print("2. Create nginx/default.conf for API gateway")
    print("3. Update docker-compose.yml with all services")
