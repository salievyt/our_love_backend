from django.utils.timezone import now
from rest_framework import generics, status
from rest_framework.decorators import api_view
from rest_framework.response import Response

from apps.questions.models import Answer, QuestionOfTheDay, QuestionTemplate
from apps.questions.serializers import (
    AnswerCreateSerializer,
    AnswerSerializer,
    QuestionOfTheDaySerializer,
    QuestionTemplateSerializer,
)


class TodayQuestionView(generics.RetrieveAPIView):
    serializer_class = QuestionOfTheDaySerializer

    def get_object(self):
        today = now().date()
        question, created = QuestionOfTheDay.objects.get_or_create(
            date_assigned=today,
            defaults={
                "question": "Что ты больше всего ценишь в наших отношениях?",
            },
        )
        if created:
            template = QuestionTemplate.objects.order_by("?").first()
            if template:
                question.question = template.text
                question.save()
        return question


class QuestionListCreateView(generics.ListCreateAPIView):
    serializer_class = QuestionOfTheDaySerializer

    def get_queryset(self):
        return QuestionOfTheDay.objects.all().order_by("-date_assigned")


class QuestionDetailView(generics.RetrieveAPIView):
    serializer_class = QuestionOfTheDaySerializer
    queryset = QuestionOfTheDay.objects.all()


class AnswerCreateView(generics.CreateAPIView):
    serializer_class = AnswerCreateSerializer

    def perform_create(self, serializer):
        serializer.save(user=self.request.user)


class QuestionAnswerListView(generics.ListAPIView):
    serializer_class = AnswerSerializer

    def get_queryset(self):
        return Answer.objects.filter(question_id=self.kwargs["question_pk"])


@api_view(["GET"])
def generate_question_view(request):
    today = now().date()
    if QuestionOfTheDay.objects.filter(date_assigned=today).exists():
        return Response(
            {"detail": "Вопрос на сегодня уже существует"},
            status=status.HTTP_400_BAD_REQUEST,
        )

    template = QuestionTemplate.objects.order_by("?").first()
    if not template:
        return Response(
            {"detail": "Нет доступных шаблонов вопросов"},
            status=status.HTTP_400_BAD_REQUEST,
        )

    question = QuestionOfTheDay.objects.create(
        question=template.text,
        date_assigned=today,
    )

    return Response(
        QuestionOfTheDaySerializer(question, context={"request": request}).data,
        status=status.HTTP_201_CREATED,
    )


class QuestionTemplateListView(generics.ListAPIView):
    queryset = QuestionTemplate.objects.all()
    serializer_class = QuestionTemplateSerializer
