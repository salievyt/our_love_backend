from django.urls import path

from apps.tasks import views

urlpatterns = [
    path("", views.DailyTaskListCreateView.as_view(), name="task-list"),
    path("today/", views.TodayTaskView.as_view(), name="task-today"),
    path("generate/", views.generate_task_view, name="task-generate"),
    path("<uuid:pk>/", views.DailyTaskDetailView.as_view(), name="task-detail"),
    path("<uuid:pk>/complete/", views.CompleteTaskView.as_view(), name="task-complete"),
    path("templates/", views.TaskTemplateListView.as_view(), name="task-templates"),
]
