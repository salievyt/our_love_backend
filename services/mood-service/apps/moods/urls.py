from django.urls import path

from apps.moods import views

urlpatterns = [
    path("", views.MoodEntryListCreateView.as_view(), name="mood-list"),
    path("today/", views.TodayMoodView.as_view(), name="mood-today"),
    path("stats/", views.MoodStatsView.as_view(), name="mood-stats"),
    path("<uuid:pk>/", views.MoodEntryDetailView.as_view(), name="mood-detail"),
]
