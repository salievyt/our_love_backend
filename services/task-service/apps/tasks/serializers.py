from rest_framework import serializers

from apps.tasks.models import DailyTask, TaskTemplate


class TaskTemplateSerializer(serializers.ModelSerializer):
    class Meta:
        model = TaskTemplate
        fields = ("id", "title", "description", "emoji")


class DailyTaskSerializer(serializers.ModelSerializer):
    photo_url = serializers.SerializerMethodField()

    class Meta:
        model = DailyTask
        fields = (
            "id",
            "title",
            "description",
            "emoji",
            "is_completed",
            "completed_at",
            "photo",
            "photo_url",
            "date_assigned",
            "created_at",
        )
        read_only_fields = ("id", "created_at")

    def get_photo_url(self, obj):
        if obj.photo:
            return obj.photo.url
        return None
