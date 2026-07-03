from rest_framework import serializers

from apps.dates.models import ImportantDate


class ImportantDateSerializer(serializers.ModelSerializer):
    days_until = serializers.IntegerField(read_only=True)

    class Meta:
        model = ImportantDate
        fields = (
            "id",
            "relationship",
            "title",
            "date",
            "emoji",
            "is_annually",
            "days_until",
            "created_at",
        )
        read_only_fields = ("id", "days_until", "created_at")


class ImportantDateCreateSerializer(serializers.ModelSerializer):
    class Meta:
        model = ImportantDate
        fields = ("title", "date", "emoji", "is_annually")

    def create(self, validated_data):
        from apps.accounts.models import Relationship

        user = self.context["request"].user
        relationship = (
            getattr(user, "relationship_as_partner1", None)
            or getattr(user, "relationship_as_partner2", None)
        )
        validated_data["relationship"] = relationship
        validated_data["created_by"] = user
        return super().create(validated_data)
