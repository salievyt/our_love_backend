from django.contrib.auth import get_user_model
from rest_framework import serializers

from apps.accounts.models import InviteCode, Relationship

User = get_user_model()


class UserSerializer(serializers.ModelSerializer):
    class Meta:
        model = User
        fields = ("id", "username", "email", "first_name", "last_name", "avatar", "phone")
        read_only_fields = ("id",)


class RegisterSerializer(serializers.ModelSerializer):
    password = serializers.CharField(write_only=True, min_length=8)
    invite_code = serializers.CharField(write_only=True, required=False, allow_blank=True)

    class Meta:
        model = User
        fields = ("username", "email", "password", "first_name", "last_name", "invite_code")

    def create(self, validated_data):
        validated_data.pop("invite_code", None)
        password = validated_data.pop("password")
        user = User(**validated_data)
        user.set_password(password)
        user.save()
        return user


class RelationshipSerializer(serializers.ModelSerializer):
    partner1 = UserSerializer(read_only=True)
    partner2 = UserSerializer(read_only=True)
    days_together = serializers.IntegerField(read_only=True)

    class Meta:
        model = Relationship
        fields = (
            "id",
            "partner1",
            "partner2",
            "partner1_name",
            "partner2_name",
            "partner1_emoji",
            "partner2_emoji",
            "relationship_start_date",
            "days_together",
            "created_at",
            "updated_at",
        )
        read_only_fields = ("id", "created_at", "updated_at", "days_together")


class InviteCodeSerializer(serializers.ModelSerializer):
    class Meta:
        model = InviteCode
        fields = ("id", "code", "is_used", "created_at", "expires_at")
        read_only_fields = ("id", "code", "is_used", "created_at", "expires_at")


class InviteCodeGenerateSerializer(serializers.Serializer):
    pass
