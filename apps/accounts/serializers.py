from django.contrib.auth import get_user_model
from django.contrib.auth.password_validation import validate_password
from rest_framework import serializers

from apps.accounts.models import GENDER_CHOICES, InviteCode, PartnerLike, PartnerProfile, Relationship

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

    def validate(self, attrs):
        # Полный набор валидаторов Django, включая похожесть на логин и словарь частых паролей
        candidate = User(**{k: v for k, v in attrs.items() if k not in ("password", "invite_code")})
        try:
            validate_password(attrs["password"], user=candidate)
        except Exception as exc:  # django.core.exceptions.ValidationError
            raise serializers.ValidationError({"password": list(exc.messages)})
        return attrs

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


class InviteCodeAcceptSerializer(serializers.Serializer):
    invite_code = serializers.CharField(max_length=32)


class PartnerProfileSerializer(serializers.ModelSerializer):
    age = serializers.IntegerField(read_only=True)

    class Meta:
        model = PartnerProfile
        fields = (
            "id", "display_name", "bio", "city",
            "birth_date", "age", "gender", "photo",
            "is_active", "created_at", "updated_at",
        )
        read_only_fields = ("id", "age", "created_at", "updated_at")

    def validate_birth_date(self, value):
        from datetime import date
        if value and value > date.today():
            raise serializers.ValidationError("Дата рождения не может быть в будущем")
        return value


class PartnerProfileCreateSerializer(serializers.ModelSerializer):
    class Meta:
        model = PartnerProfile
        fields = ("display_name", "bio", "city", "birth_date", "gender", "photo")

    def create(self, validated_data):
        user = self.context["request"].user
        profile, created = PartnerProfile.objects.update_or_create(
            user=user,
            defaults={
                **validated_data,
                "is_active": True,
            },
        )
        return profile


class PartnerSearchSerializer(serializers.Serializer):
    city = serializers.CharField(required=False, allow_blank=True)
    gender = serializers.ChoiceField(choices=GENDER_CHOICES, required=False)
    min_age = serializers.IntegerField(required=False, min_value=18)
    max_age = serializers.IntegerField(required=False, min_value=18)
    exclude_liked = serializers.BooleanField(default=True)


class PartnerLikeSerializer(serializers.ModelSerializer):
    class Meta:
        model = PartnerLike
        fields = ("id", "to_user", "is_like", "created_at")
        read_only_fields = ("id", "created_at")

    def validate_to_user(self, value):
        user = self.context["request"].user
        if value == user:
            raise serializers.ValidationError("Нельзя оценить свою анкету")
        if not hasattr(value, "partner_profile") or not value.partner_profile.is_active:
            raise serializers.ValidationError("У пользователя нет активной анкеты")
        return value
