from django.utils.timezone import now
from rest_framework import generics, status
from rest_framework.decorators import api_view
from rest_framework.response import Response

from apps.tasks.models import DailyTask, TaskTemplate
from apps.tasks.serializers import DailyTaskSerializer, TaskTemplateSerializer


class TodayTaskView(generics.RetrieveAPIView):
    serializer_class = DailyTaskSerializer

    def get_object(self):
        today = now().date()
        task, created = DailyTask.objects.get_or_create(
            date_assigned=today,
            defaults={
                "title": "Отдохните вместе",
                "description": "Проведите время вместе, наслаждаясь моментом",
                "emoji": "💕",
            },
        )
        if created:
            template = TaskTemplate.objects.order_by("?").first()
            if template:
                task.title = template.title
                task.description = template.description
                task.emoji = template.emoji
                task.save()
        return task


class DailyTaskListCreateView(generics.ListCreateAPIView):
    serializer_class = DailyTaskSerializer
    filterset_fields = ("is_completed", "date_assigned")

    def get_queryset(self):
        return DailyTask.objects.all().order_by("-date_assigned")

    def perform_create(self, serializer):
        serializer.save(date_assigned=now().date())


class DailyTaskDetailView(generics.RetrieveUpdateDestroyAPIView):
    serializer_class = DailyTaskSerializer

    def get_queryset(self):
        return DailyTask.objects.all()


class CompleteTaskView(generics.UpdateAPIView):
    serializer_class = DailyTaskSerializer

    def get_queryset(self):
        return DailyTask.objects.all()

    def perform_update(self, serializer):
        serializer.save(is_completed=True, completed_at=now())


@api_view(["GET"])
def generate_task_view(request):
    today = now().date()
    if DailyTask.objects.filter(date_assigned=today).exists():
        return Response(
            {"detail": "Задание на сегодня уже существует"},
            status=status.HTTP_400_BAD_REQUEST,
        )

    template = TaskTemplate.objects.order_by("?").first()
    if not template:
        return Response(
            {"detail": "Нет доступных шаблонов заданий"},
            status=status.HTTP_400_BAD_REQUEST,
        )

    task = DailyTask.objects.create(
        title=template.title,
        description=template.description,
        emoji=template.emoji,
        date_assigned=today,
    )

    return Response(DailyTaskSerializer(task).data, status=status.HTTP_201_CREATED)


class TaskTemplateListView(generics.ListAPIView):
    queryset = TaskTemplate.objects.all()
    serializer_class = TaskTemplateSerializer
