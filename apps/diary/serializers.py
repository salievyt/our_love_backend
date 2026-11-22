from rest_framework import serializers

from config.media import file_url

from apps.diary.models import DiaryEntry


class DiaryEntrySerializer(serializers.ModelSerializer):
    photo_url = serializers.SerializerMethodField()

    class Meta:
        model = DiaryEntry
        fields = (
            "id",
            "date",
            "note",
            "photo",
            "photo_url",
            "mood_partner1",
            "mood_partner2",
            "question_answer_partner1",
            "question_answer_partner2",
            "created_at",
            "updated_at",
        )
        read_only_fields = (
            "id",
            "question_answer_partner1",
            "question_answer_partner2",
            "created_at",
            "updated_at",
        )

    def validate_date(self, value):
        entry = self.instance
        if (
            entry is not None
            and DiaryEntry.objects.filter(relationship=entry.relationship, date=value)
            .exclude(pk=entry.pk)
            .exists()
        ):
            raise serializers.ValidationError("Запись на этот день уже существует")
        return value

    def get_photo_url(self, obj):
        return file_url(obj.photo, self.context)


class DiaryEntryCreateSerializer(serializers.ModelSerializer):
    class Meta:
        model = DiaryEntry
        fields = ("date", "note", "photo", "mood_partner1")

    def create(self, validated_data):
        if DiaryEntry.objects.filter(
            relationship=validated_data["relationship"], date=validated_data.get("date")
        ).exists():
            raise serializers.ValidationError({"date": "Запись на этот день уже существует"})
        return super().create(validated_data)
