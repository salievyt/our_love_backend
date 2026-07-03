from rest_framework import serializers

from apps.albums.models import Collage, Photo


class PhotoSerializer(serializers.ModelSerializer):
    image_url = serializers.SerializerMethodField()
    upload_url = serializers.URLField(write_only=True, required=False)

    class Meta:
        model = Photo
        fields = (
            "id",
            "image",
            "image_url",
            "caption",
            "created_at",
            "year_tag",
            "latitude",
            "longitude",
            "upload_url",
        )
        read_only_fields = ("id", "created_at", "year_tag")

    def get_image_url(self, obj):
        if obj.image:
            return obj.image.url
        return None


class CollageSerializer(serializers.ModelSerializer):
    photo_count = serializers.IntegerField(source="photos.count", read_only=True)

    class Meta:
        model = Collage
        fields = ("id", "title", "image", "photos", "photo_count", "created_at")
        read_only_fields = ("id", "created_at", "photo_count")
