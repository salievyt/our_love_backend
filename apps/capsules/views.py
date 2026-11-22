from django.shortcuts import get_object_or_404
from django.utils.timezone import now
from rest_framework import generics, status
from rest_framework.parsers import FormParser, MultiPartParser
from rest_framework.response import Response

from apps.accounts.scoping import pair_q
from apps.capsules.models import TimeCapsule
from apps.capsules.serializers import (
    TimeCapsuleCreateSerializer,
    TimeCapsuleOpenSerializer,
    TimeCapsuleSerializer,
)


def pair_capsules(user):
    """Капсулы, созданные пользователем или его партнёром."""
    return TimeCapsule.objects.filter(pair_q(user, "created_by"))


class TimeCapsuleListCreateView(generics.ListCreateAPIView):
    parser_classes = (MultiPartParser, FormParser)

    def get_serializer_class(self):
        if self.request.method == "POST":
            return TimeCapsuleCreateSerializer
        return TimeCapsuleSerializer

    def get_queryset(self):
        return pair_capsules(self.request.user).order_by("open_date")

    def perform_create(self, serializer):
        serializer.save(created_by=self.request.user)


class TimeCapsuleDetailView(generics.RetrieveUpdateDestroyAPIView):
    parser_classes = (MultiPartParser, FormParser)

    def get_serializer_class(self):
        if self.request.method in ("PUT", "PATCH"):
            return TimeCapsuleCreateSerializer
        return TimeCapsuleSerializer

    def get_queryset(self):
        qs = pair_capsules(self.request.user)
        if self.request.method in ("PUT", "PATCH"):
            # Менять можно только свою ещё не открытую капсулу
            return qs.filter(created_by=self.request.user, is_opened=False)
        if self.request.method == "DELETE":
            return qs.filter(created_by=self.request.user)
        return qs


class TimeCapsuleOpenView(generics.CreateAPIView):
    serializer_class = TimeCapsuleOpenSerializer

    def create(self, request, *args, **kwargs):
        capsule = get_object_or_404(pair_capsules(request.user), pk=self.kwargs["pk"])
        if not capsule.is_ready_to_open:
            return Response(
                {"detail": "Капсула ещё не готова к открытию"},
                status=status.HTTP_400_BAD_REQUEST,
            )
        capsule.is_opened = True
        capsule.opened_at = now()
        capsule.save(update_fields=["is_opened", "opened_at"])
        return Response(TimeCapsuleSerializer(capsule, context=self.get_serializer_context()).data)


class ActiveTimeCapsuleListView(generics.ListAPIView):
    serializer_class = TimeCapsuleSerializer

    def get_queryset(self):
        return pair_capsules(self.request.user).filter(is_opened=False).order_by("open_date")


class OpenedTimeCapsuleListView(generics.ListAPIView):
    serializer_class = TimeCapsuleSerializer

    def get_queryset(self):
        return pair_capsules(self.request.user).filter(is_opened=True).order_by("-opened_at")
