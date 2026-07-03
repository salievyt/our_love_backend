from django.urls import path

from apps.dates import views

urlpatterns = [
    path("", views.ImportantDateListCreateView.as_view(), name="date-list"),
    path("upcoming/", views.UpcomingImportantDateListView.as_view(), name="date-upcoming"),
    path("<uuid:pk>/", views.ImportantDateDetailView.as_view(), name="date-detail"),
]
