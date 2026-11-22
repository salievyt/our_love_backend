from django.shortcuts import get_object_or_404
from django.utils.timezone import now
from rest_framework import generics, status
from rest_framework.decorators import api_view
from rest_framework.exceptions import ValidationError
from rest_framework.response import Response

from apps.accounts.scoping import get_relationship, require_relationship
from apps.questions.models import Answer, QuestionOfTheDay, QuestionTemplate
from apps.questions.serializers import (
    AnswerCreateSerializer,
    AnswerSerializer,
    QuestionOfTheDaySerializer,
    QuestionTemplateSerializer,
    visible_answers,
)

DEFAULT_QUESTION = "Что ты больше всего ценишь в наших отношениях?"


def pair_questions(user):
    relationship = get_relationship(user)
    if relationship is None:
        return QuestionOfTheDay.objects.none()
    return QuestionOfTheDay.objects.filter(relationship=relationship)


def random_question_text():
    template = QuestionTemplate.objects.order_by("?").first()
    return template.text if template else DEFAULT_QUESTION


class TodayQuestionView(generics.RetrieveAPIView):
    serializer_class = QuestionOfTheDaySerializer

    def get_object(self):
        relationship = require_relationship(self.request.user)
        question = QuestionOfTheDay.objects.filter(
            relationship=relationship, date_assigned=now().date()
        ).first()
        if question is None:
            question = QuestionOfTheDay.objects.create(
                relationship=relationship,
                date_assigned=now().date(),
                question=random_question_text(),
            )
        return question


class QuestionListCreateView(generics.ListCreateAPIView):
    serializer_class = QuestionOfTheDaySerializer

    def get_queryset(self):
        return pair_questions(self.request.user).order_by("-date_assigned")

    def get_serializer_context(self):
        context = super().get_serializer_context()
        context["relationship"] = get_relationship(self.request.user)
        return context

    def perform_create(self, serializer):
        serializer.save(relationship=require_relationship(self.request.user))


class QuestionDetailView(generics.RetrieveAPIView):
    serializer_class = QuestionOfTheDaySerializer

    def get_queryset(self):
        return pair_questions(self.request.user)


class AnswerCreateView(generics.CreateAPIView):
    serializer_class = AnswerCreateSerializer

    def perform_create(self, serializer):
        question = get_object_or_404(
            pair_questions(self.request.user), pk=self.kwargs["question_pk"]
        )
        if Answer.objects.filter(question=question, user=self.request.user).exists():
            raise ValidationError({"detail": "Вы уже ответили на этот вопрос"})
        serializer.save(user=self.request.user, question=question)

    def create(self, request, *args, **kwargs):
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        self.perform_create(serializer)
        return Response(AnswerSerializer(serializer.instance).data, status=status.HTTP_201_CREATED)


class QuestionAnswerListView(generics.ListAPIView):
    serializer_class = AnswerSerializer

    def get_queryset(self):
        question = get_object_or_404(
            pair_questions(self.request.user), pk=self.kwargs["question_pk"]
        )
        return visible_answers(question, self.request.user)


@api_view(["GET", "POST"])
def generate_question_view(request):
    relationship = require_relationship(request.user)
    today = now().date()
    if QuestionOfTheDay.objects.filter(relationship=relationship, date_assigned=today).exists():
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
        relationship=relationship,
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
