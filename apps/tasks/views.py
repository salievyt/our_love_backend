from django.utils.timezone import now
from rest_framework import generics, status
from rest_framework.decorators import api_view
from rest_framework.response import Response

from apps.accounts.scoping import get_relationship, require_relationship
from apps.tasks.models import DailyTask, TaskTemplate
from apps.tasks.serializers import DailyTaskSerializer, TaskTemplateSerializer


def pair_tasks(user):
    relationship = get_relationship(user)
    if relationship is None:
        return DailyTask.objects.none()
    return DailyTask.objects.filter(relationship=relationship)


def create_task_from_template(relationship, date):
    template = TaskTemplate.objects.order_by("?").first()
    if template:
        fields = {
            "title": template.title,
            "description": template.description,
            "emoji": template.emoji,
        }
    else:
        fields = {
            "title": "Отдохните вместе",
            "description": "Проведите время вместе, наслаждаясь моментом",
            "emoji": "💕",
        }
    return DailyTask.objects.create(relationship=relationship, date_assigned=date, **fields)


class TodayTaskView(generics.RetrieveAPIView):
    serializer_class = DailyTaskSerializer

    def get_object(self):
        relationship = require_relationship(self.request.user)
        today = now().date()
        task = (
            DailyTask.objects.filter(relationship=relationship, date_assigned=today)
            .order_by("created_at")
            .first()
        )
        return task or create_task_from_template(relationship, today)


class DailyTaskListCreateView(generics.ListCreateAPIView):
    serializer_class = DailyTaskSerializer
    filterset_fields = ("is_completed", "date_assigned")

    def get_queryset(self):
        return pair_tasks(self.request.user).order_by("-date_assigned")

    def perform_create(self, serializer):
        serializer.save(
            relationship=require_relationship(self.request.user),
            date_assigned=serializer.validated_data.get("date_assigned", now().date()),
        )


class DailyTaskDetailView(generics.RetrieveUpdateDestroyAPIView):
    serializer_class = DailyTaskSerializer

    def get_queryset(self):
        return pair_tasks(self.request.user)


class CompleteTaskView(generics.UpdateAPIView):
    serializer_class = DailyTaskSerializer

    def get_queryset(self):
        return pair_tasks(self.request.user)

    def update(self, request, *args, **kwargs):
        # Тело запроса необязательно: iOS шлёт пустой PUT
        kwargs["partial"] = True
        return super().update(request, *args, **kwargs)

    def perform_update(self, serializer):
        serializer.save(is_completed=True, completed_at=now())

    def post(self, request, *args, **kwargs):
        # Android-клиент отмечает выполнение через POST
        return self.update(request, *args, **kwargs)


@api_view(["GET", "POST"])
def generate_task_view(request):
    relationship = require_relationship(request.user)
    today = now().date()
    if DailyTask.objects.filter(relationship=relationship, date_assigned=today).exists():
        return Response(
            {"detail": "Задание на сегодня уже существует"},
            status=status.HTTP_400_BAD_REQUEST,
        )

    if not TaskTemplate.objects.exists():
        return Response(
            {"detail": "Нет доступных шаблонов заданий"},
            status=status.HTTP_400_BAD_REQUEST,
        )

    task = create_task_from_template(relationship, today)
    return Response(
        DailyTaskSerializer(task, context={"request": request}).data,
        status=status.HTTP_201_CREATED,
    )


class TaskTemplateListView(generics.ListAPIView):
    queryset = TaskTemplate.objects.all()
    serializer_class = TaskTemplateSerializer
