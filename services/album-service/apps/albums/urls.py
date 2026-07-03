from django.urls import path

from apps.albums import views

urlpatterns = [
    path("", views.PhotoListCreateView.as_view(), name="photo-list"),
    path("<uuid:pk>/", views.PhotoDetailView.as_view(), name="photo-detail"),
    path("collages/", views.CollageListCreateView.as_view(), name="collage-list"),
    path("collages/<uuid:pk>/", views.CollageDetailView.as_view(), name="collage-detail"),
]
