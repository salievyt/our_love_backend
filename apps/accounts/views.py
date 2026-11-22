from django.contrib.auth import get_user_model
from django.db import transaction
from rest_framework import generics, permissions, status
from rest_framework.decorators import api_view, permission_classes
from rest_framework.response import Response

from apps.accounts.invites import accept_invite, issue_invite

from django.db.models import Q

from apps.accounts.models import InviteCode, PartnerLike, PartnerProfile, Relationship
from apps.accounts.serializers import (
    InviteCodeAcceptSerializer,
    InviteCodeGenerateSerializer,
    InviteCodeSerializer,
    PartnerLikeSerializer,
    PartnerProfileCreateSerializer,
    PartnerProfileSerializer,
    RegisterSerializer,
    RelationshipSerializer,
    UserSerializer,
)

User = get_user_model()


class RegisterView(generics.CreateAPIView):
    queryset = User.objects.all()
    permission_classes = (permissions.AllowAny,)
    serializer_class = RegisterSerializer

    def create(self, request, *args, **kwargs):
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        code = serializer.validated_data.get("invite_code")

        # Пользователь и пара создаются вместе: при неверном коде не остаётся ни того, ни другого
        with transaction.atomic():
            user = serializer.save()
            if code:
                accept_invite(user, code)

        return Response(
            UserSerializer(user).data,
            status=status.HTTP_201_CREATED,
        )


class UserProfileView(generics.RetrieveUpdateAPIView):
    serializer_class = UserSerializer

    def get_object(self):
        return self.request.user


class RelationshipDetailView(generics.RetrieveUpdateAPIView):
    serializer_class = RelationshipSerializer

    def get_object(self):
        return getattr(self.request.user, "relationship_as_partner1", None) or getattr(
            self.request.user, "relationship_as_partner2", None
        )


class GenerateInviteCodeView(generics.CreateAPIView):
    serializer_class = InviteCodeGenerateSerializer

    def create(self, request, *args, **kwargs):
        invite = issue_invite(request.user)
        return Response(InviteCodeSerializer(invite).data, status=status.HTTP_201_CREATED)


class AcceptInviteCodeView(generics.CreateAPIView):
    """Для уже зарегистрированного пользователя: создать пару по коду партнёра."""

    serializer_class = InviteCodeAcceptSerializer

    def create(self, request, *args, **kwargs):
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        with transaction.atomic():
            relationship = accept_invite(request.user, serializer.validated_data["invite_code"])
        return Response(RelationshipSerializer(relationship).data, status=status.HTTP_201_CREATED)


@api_view(["GET"])
@permission_classes((permissions.IsAuthenticated,))
def me_view(request):
    return Response(UserSerializer(request.user).data)


class PartnerProfileView(generics.RetrieveUpdateAPIView):
    """Получить / обновить свою анкету для поиска"""
    serializer_class = PartnerProfileSerializer

    def get_object(self):
        return getattr(self.request.user, "partner_profile", None)

    def get(self, request, *args, **kwargs):
        profile = self.get_object()
        if not profile:
            return Response(
                {"detail": "Анкета не создана"},
                status=status.HTTP_404_NOT_FOUND,
            )
        return Response(self.get_serializer(profile).data)


class PartnerProfileCreateView(generics.CreateAPIView):
    """Создать анкету для поиска"""
    serializer_class = PartnerProfileCreateSerializer

    def create(self, request, *args, **kwargs):
        if hasattr(request.user, "partner_profile"):
            # Update existing
            serializer = self.get_serializer(data=request.data, partial=True)
            serializer.is_valid(raise_exception=True)
            profile = request.user.partner_profile
            for key, value in serializer.validated_data.items():
                setattr(profile, key, value)
            profile.is_active = True
            profile.save()
            return Response(
                PartnerProfileSerializer(profile).data,
                status=status.HTTP_200_OK,
            )
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        profile = serializer.save()
        return Response(
            PartnerProfileSerializer(profile).data,
            status=status.HTTP_201_CREATED,
        )


class PartnerProfileDeleteView(generics.DestroyAPIView):
    """Удалить анкету (деактивировать)"""

    def delete(self, request, *args, **kwargs):
        profile = getattr(request.user, "partner_profile", None)
        if profile:
            profile.is_active = False
            profile.save()
        return Response(status=status.HTTP_204_NO_CONTENT)


class PartnerSearchView(generics.ListAPIView):
    """Поиск анкет для поиска партнёра"""
    serializer_class = PartnerProfileSerializer

    def get_queryset(self):
        user = self.request.user
        qs = PartnerProfile.objects.filter(is_active=True).exclude(user=user)

        # City filter
        city = self.request.query_params.get("city", "")
        if city:
            qs = qs.filter(city__icontains=city)

        # Gender filter
        gender = self.request.query_params.get("gender", "")
        if gender:
            qs = qs.filter(gender=gender)

        # Age filter (from birth_date)
        from datetime import date, timedelta
        min_age = self.request.query_params.get("min_age")
        max_age = self.request.query_params.get("max_age")
        today = date.today()
        if max_age:
            try:
                max_date = date(today.year - int(max_age), today.month, today.day)
                qs = qs.filter(birth_date__gte=max_date)
            except (ValueError, OverflowError):
                pass
        if min_age:
            try:
                min_date = date(today.year - int(min_age), today.month, today.day)
                qs = qs.filter(birth_date__lte=min_date)
            except (ValueError, OverflowError):
                pass

        # Exclude already liked/passed
        exclude_liked = self.request.query_params.get("exclude_liked", "true").lower() == "true"
        if exclude_liked:
            liked_user_ids = PartnerLike.objects.filter(
                from_user=user
            ).values_list("to_user_id", flat=True)
            qs = qs.exclude(user_id__in=liked_user_ids)

        # Exclude users already in a relationship (subquery for performance)
        from django.contrib.auth import get_user_model
        User = get_user_model()
        in_rel_qs = User.objects.filter(
            Q(relationship_as_partner1__isnull=False) |
            Q(relationship_as_partner2__isnull=False)
        ).values('id')
        qs = qs.exclude(user_id__in=in_rel_qs)

        return qs.order_by("?")  # Random order


class PartnerLikeView(generics.CreateAPIView):
    """Поставить лайк / пропуск"""
    serializer_class = PartnerLikeSerializer

    def create(self, request, *args, **kwargs):
        to_user_id = request.data.get("to_user")
        is_like = request.data.get("is_like", True)
        user = request.user

        from django.contrib.auth import get_user_model
        User = get_user_model()
        try:
            to_user = User.objects.get(id=to_user_id)
        except User.DoesNotExist:
            return Response(
                {"detail": "Пользователь не найден"},
                status=status.HTTP_404_NOT_FOUND,
            )

        if to_user == user:
            return Response(
                {"detail": "Нельзя оценить себя"},
                status=status.HTTP_400_BAD_REQUEST,
            )

        like, created = PartnerLike.objects.update_or_create(
            from_user=user,
            to_user=to_user,
            defaults={"is_like": is_like},
        )

        # Check for mutual like
        is_match = False
        if is_like:
            mutual = PartnerLike.objects.filter(
                from_user=to_user,
                to_user=user,
                is_like=True,
            ).exists()
            if mutual:
                is_match = True

        return Response({
            "like": PartnerLikeSerializer(like).data,
            "is_match": is_match,
        }, status=status.HTTP_201_CREATED if created else status.HTTP_200_OK)


class PartnerMatchesView(generics.ListAPIView):
    """Список взаимных лайков"""
    serializer_class = PartnerProfileSerializer

    def get_queryset(self):
        user = self.request.user
        liked_me = PartnerLike.objects.filter(
            to_user=user, is_like=True
        ).values_list("from_user_id", flat=True)
        i_liked = PartnerLike.objects.filter(
            from_user=user, is_like=True, to_user_id__in=liked_me
        ).values_list("to_user_id", flat=True)
        return PartnerProfile.objects.filter(
            user_id__in=set(liked_me) & set(i_liked),
            is_active=True,
        )
