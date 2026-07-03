from rest_framework import serializers

from apps.capsules.models import TimeCapsule


class TimeCapsuleSerializer(serializers.ModelSerializer):
    photo_url = serializers.SerializerMethodField()
    voice_note_url = serializers.SerializerMethodField()
    is_ready_to_open = serializers.BooleanField(read_only=True)
    days_until_open = serializers.SerializerMethodField()

    class Meta:
        model = TimeCapsule
        fields = (
            "id",
            "title",
            "message",
            "photo",
            "photo_url",
            "voice_note",
            "voice_note_url",
            "open_date",
            "is_opened",
            "is_ready_to_open",
            "days_until_open",
            "opened_at",
            "created_at",
        )
        read_only_fields = ("id", "is_opened", "is_ready_to_open", "opened_at", "created_at")

    def get_photo_url(self, obj):
        if obj.photo:
            return obj.photo.url
        return None

    def get_voice_note_url(self, obj):
        if obj.voice_note:
            return obj.voice_note.url
        return None

    def get_days_until_open(self, obj):
        from django.utils.timezone import now

        delta = obj.open_date - now().date()
        return max(delta.days, 0)


class TimeCapsuleCreateSerializer(serializers.ModelSerializer):
    class Meta:
        model = TimeCapsule
        fields = ("title", "message", "photo", "voice_note", "open_date")

    def create(self, validated_data):
        validated_data["created_by"] = self.context["request"].user
        return super().create(validated_data)


class TimeCapsuleOpenSerializer(serializers.Serializer):
    confirm = serializers.BooleanField(default=True)

    def save(self, **kwargs):
        capsule = self.context["capsule"]
        if capsule.is_ready_to_open:
            from django.utils.timezone import now

            capsule.is_opened = True
            capsule.opened_at = now()
            capsule.save()
            return capsule
        raise serializers.ValidationError("Капсула ещё не готова к открытию")
