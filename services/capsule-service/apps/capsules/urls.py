from django.urls import path

from apps.capsules import views

urlpatterns = [
    path("", views.TimeCapsuleListCreateView.as_view(), name="capsule-list"),
    path("active/", views.ActiveTimeCapsuleListView.as_view(), name="capsule-active"),
    path("opened/", views.OpenedTimeCapsuleListView.as_view(), name="capsule-opened"),
    path("<uuid:pk>/", views.TimeCapsuleDetailView.as_view(), name="capsule-detail"),
    path("<uuid:pk>/open/", views.TimeCapsuleOpenView.as_view(), name="capsule-open"),
]
