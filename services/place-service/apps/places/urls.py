from django.urls import path

from apps.places import views

urlpatterns = [
    path("", views.PlaceListCreateView.as_view(), name="place-list"),
    path("<uuid:pk>/", views.PlaceDetailView.as_view(), name="place-detail"),
]
