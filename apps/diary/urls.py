from django.urls import path

from apps.diary import views

urlpatterns = [
    path("", views.DiaryEntryListCreateView.as_view(), name="diary-list"),
    path("today/", views.TodayDiaryEntryView.as_view(), name="diary-today"),
    path("<uuid:pk>/", views.DiaryEntryDetailView.as_view(), name="diary-detail"),
]
