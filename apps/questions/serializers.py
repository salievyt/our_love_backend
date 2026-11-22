from rest_framework import serializers

from apps.questions.models import Answer, QuestionOfTheDay, QuestionTemplate


class QuestionTemplateSerializer(serializers.ModelSerializer):
    class Meta:
        model = QuestionTemplate
        fields = ("id", "text")


class AnswerSerializer(serializers.ModelSerializer):
    class Meta:
        model = Answer
        fields = ("id", "question", "user", "text", "created_at")
        read_only_fields = ("id", "question", "user", "created_at")


class AnswerCreateSerializer(serializers.ModelSerializer):
    class Meta:
        model = Answer
        fields = ("text",)


def visible_answers(question, user):
    """Ответы пары на вопрос. Ответ партнёра виден только после своего ответа."""
    relationship = question.relationship
    if relationship is None:
        return Answer.objects.none()
    answers = question.answers.filter(
        user_id__in=(relationship.partner1_id, relationship.partner2_id)
    )
    if not answers.filter(user=user).exists():
        return answers.none()
    return answers.order_by("created_at")


class QuestionOfTheDaySerializer(serializers.ModelSerializer):
    answers = serializers.SerializerMethodField()
    partner1_answered = serializers.SerializerMethodField()
    partner2_answered = serializers.SerializerMethodField()
    both_answered = serializers.SerializerMethodField()

    class Meta:
        model = QuestionOfTheDay
        fields = (
            "id",
            "question",
            "date_assigned",
            "answers",
            "partner1_answered",
            "partner2_answered",
            "both_answered",
            "created_at",
        )
        read_only_fields = ("id", "created_at")

    def validate_date_assigned(self, value):
        relationship = self.context.get("relationship")
        qs = QuestionOfTheDay.objects.filter(relationship=relationship, date_assigned=value)
        if self.instance is not None:
            qs = qs.exclude(pk=self.instance.pk)
        if relationship is not None and qs.exists():
            raise serializers.ValidationError("Вопрос на этот день уже существует")
        return value

    def get_answers(self, obj):
        request = self.context.get("request")
        if request is None:
            return []
        return AnswerSerializer(visible_answers(obj, request.user), many=True).data

    def get_partner1_answered(self, obj):
        return self._has_answer(obj, "partner1_id")

    def get_partner2_answered(self, obj):
        return self._has_answer(obj, "partner2_id")

    def get_both_answered(self, obj):
        return self.get_partner1_answered(obj) and self.get_partner2_answered(obj)

    def _has_answer(self, obj, partner_field):
        if obj.relationship is None:
            return False
        return obj.answers.filter(user_id=getattr(obj.relationship, partner_field)).exists()
