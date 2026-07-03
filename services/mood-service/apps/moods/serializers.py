from rest_framework import serializers

from apps.moods.models import MoodEntry


class MoodEntrySerializer(serializers.ModelSerializer):
    class Meta:
        model = MoodEntry
        fields = ("id", "user", "mood", "note", "date", "created_at")
        read_only_fields = ("id", "user", "created_at")


class MoodEntryCreateSerializer(serializers.ModelSerializer):
    class Meta:
        model = MoodEntry
        fields = ("mood", "note", "date")

    def create(self, validated_data):
        validated_data["user"] = self.context["request"].user
        return super().create(validated_data)
