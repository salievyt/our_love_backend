from rest_framework import serializers

from apps.places.models import Place


class PlaceSerializer(serializers.ModelSerializer):
    photo_url = serializers.SerializerMethodField()

    class Meta:
        model = Place
        fields = (
            "id",
            "title",
            "description",
            "emoji",
            "latitude",
            "longitude",
            "photo",
            "photo_url",
            "created_at",
        )
        read_only_fields = ("id", "created_at")

    def get_photo_url(self, obj):
        if obj.photo:
            return obj.photo.url
        return None


class PlaceCreateSerializer(serializers.ModelSerializer):
    class Meta:
        model = Place
        fields = ("title", "description", "emoji", "latitude", "longitude", "photo")

    def create(self, validated_data):
        validated_data["created_by"] = self.context["request"].user
        return super().create(validated_data)
