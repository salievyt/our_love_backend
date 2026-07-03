from django.contrib.auth import get_user_model
from rest_framework import generics, permissions, status
from rest_framework.decorators import api_view, permission_classes
from rest_framework.response import Response

from apps.accounts.models import InviteCode, Relationship
from apps.accounts.serializers import (
    InviteCodeGenerateSerializer,
    InviteCodeSerializer,
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
        user = serializer.save()

        invite_code_str = request.data.get("invite_code")
        if invite_code_str:
            try:
                invite = InviteCode.objects.get(
                    code=invite_code_str, is_used=False, expires_at__gte=user.date_joined
                )
                invite.is_used = True
                invite.save()
                Relationship.objects.create(
                    partner1=invite.created_by,
                    partner2=user,
                    relationship_start_date=invite.relationship.relationship_start_date,
                    partner1_name=invite.relationship.partner1_name,
                    partner2_name=invite.relationship.partner2_name,
                )
            except InviteCode.DoesNotExist:
                pass

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
        relationship = getattr(
            request.user, "relationship_as_partner1", None
        ) or getattr(request.user, "relationship_as_partner2", None)

        if not relationship:
            return Response(
                {"detail": "У вас нет пары. Создайте отношения."},
                status=status.HTTP_400_BAD_REQUEST,
            )

        import secrets
        from datetime import timedelta

        from django.utils.timezone import now

        code = secrets.token_hex(4).upper()
        invite = InviteCode.objects.create(
            relationship=relationship,
            code=code,
            created_by=request.user,
            expires_at=now() + timedelta(days=7),
        )

        return Response(
            InviteCodeSerializer(invite).data,
            status=status.HTTP_201_CREATED,
        )


@api_view(["GET"])
@permission_classes((permissions.IsAuthenticated,))
def me_view(request):
    return Response(UserSerializer(request.user).data)
