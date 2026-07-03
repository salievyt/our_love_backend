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
        read_only_fields = ("id", "user", "created_at")


class AnswerCreateSerializer(serializers.ModelSerializer):
    class Meta:
        model = Answer
        fields = ("text",)

    def create(self, validated_data):
        validated_data["user"] = self.context["request"].user
        validated_data["question_id"] = self.context["view"].kwargs.get("question_pk")
        return super().create(validated_data)


class QuestionOfTheDaySerializer(serializers.ModelSerializer):
    answers = AnswerSerializer(many=True, read_only=True)
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

    def get_partner1_answered(self, obj):
        return self._has_answer(obj, "partner1")

    def get_partner2_answered(self, obj):
        return self._has_answer(obj, "partner2")

    def get_both_answered(self, obj):
        return self.get_partner1_answered(obj) and self.get_partner2_answered(obj)

    def _has_answer(self, obj, partner):
        user = self.context.get("request").user if "request" in self.context else None
        # Simplified: check if any answer exists for this question
        return obj.answers.exists()
