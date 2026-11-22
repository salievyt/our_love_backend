from django.contrib import admin
from django.urls import path, include
from config.media import serve_media
from drf_spectacular.views import (
    SpectacularAPIView,
    SpectacularSwaggerView,
    SpectacularRedocView,
)

api_urlpatterns = [
    path("auth/", include("apps.accounts.urls")),
    path("albums/", include("apps.albums.urls")),
    path("tasks/", include("apps.tasks.urls")),
    path("questions/", include("apps.questions.urls")),
    path("diary/", include("apps.diary.urls")),
    path("moods/", include("apps.moods.urls")),
    path("places/", include("apps.places.urls")),
    path("capsules/", include("apps.capsules.urls")),
    path("dates/", include("apps.dates.urls")),
]

urlpatterns = [
    path("admin/", admin.site.urls),
    path("api/v1/", include(api_urlpatterns)),
    path("api/schema/", SpectacularAPIView.as_view(), name="schema"),
    path("api/docs/", SpectacularSwaggerView.as_view(url_name="schema"), name="swagger-ui"),
    path("api/redoc/", SpectacularRedocView.as_view(url_name="schema"), name="redoc"),
    # Медиа только по подписанным ссылкам (см. config/media.py)
    path("media/<path:path>", serve_media, name="media"),
]
