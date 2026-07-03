from rest_framework import generics, status
from rest_framework.parsers import FormParser, MultiPartParser
from rest_framework.response import Response

from apps.capsules.models import TimeCapsule
from apps.capsules.serializers import (
    TimeCapsuleCreateSerializer,
    TimeCapsuleOpenSerializer,
    TimeCapsuleSerializer,
)


class TimeCapsuleListCreateView(generics.ListCreateAPIView):
    parser_classes = (MultiPartParser, FormParser)

    def get_serializer_class(self):
        if self.request.method == "POST":
            return TimeCapsuleCreateSerializer
        return TimeCapsuleSerializer

    def get_queryset(self):
        return TimeCapsule.objects.all().order_by("open_date")

    def perform_create(self, serializer):
        serializer.save(created_by=self.request.user)


class TimeCapsuleDetailView(generics.RetrieveUpdateDestroyAPIView):
    parser_classes = (MultiPartParser, FormParser)

    def get_serializer_class(self):
        if self.request.method in ("PUT", "PATCH"):
            return TimeCapsuleCreateSerializer
        return TimeCapsuleSerializer

    def get_queryset(self):
        return TimeCapsule.objects.all()


class TimeCapsuleOpenView(generics.CreateAPIView):
    serializer_class = TimeCapsuleOpenSerializer

    def get_serializer_context(self):
        context = super().get_serializer_context()
        capsule = TimeCapsule.objects.get(pk=self.kwargs["pk"])
        context["capsule"] = capsule
        return context

    def create(self, request, *args, **kwargs):
        capsule = TimeCapsule.objects.get(pk=self.kwargs["pk"])
        if not capsule.is_ready_to_open:
            return Response(
                {"detail": "Капсула ещё не готова к открытию"},
                status=status.HTTP_400_BAD_REQUEST,
            )
        import datetime
        from django.utils.timezone import now
        capsule.is_opened = True
        capsule.opened_at = now()
        capsule.save()
        return Response(TimeCapsuleSerializer(capsule).data)


class ActiveTimeCapsuleListView(generics.ListAPIView):
    serializer_class = TimeCapsuleSerializer

    def get_queryset(self):
        return TimeCapsule.objects.filter(is_opened=False).order_by("open_date")


class OpenedTimeCapsuleListView(generics.ListAPIView):
    serializer_class = TimeCapsuleSerializer

    def get_queryset(self):
        return TimeCapsule.objects.filter(is_opened=True).order_by("-opened_at")
