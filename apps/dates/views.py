from rest_framework import generics

from apps.dates.models import ImportantDate
from apps.dates.serializers import ImportantDateCreateSerializer, ImportantDateSerializer


class ImportantDateListCreateView(generics.ListCreateAPIView):
    filterset_fields = ("is_annually",)

    def get_serializer_class(self):
        if self.request.method == "POST":
            return ImportantDateCreateSerializer
        return ImportantDateSerializer

    def get_queryset(self):
        user = self.request.user
        relationship = (
            getattr(user, "relationship_as_partner1", None)
            or getattr(user, "relationship_as_partner2", None)
        )
        if not relationship:
            return ImportantDate.objects.none()
        return ImportantDate.objects.filter(relationship=relationship).order_by("date")

    def perform_create(self, serializer):
        serializer.save()


class ImportantDateDetailView(generics.RetrieveUpdateDestroyAPIView):
    def get_serializer_class(self):
        if self.request.method in ("PUT", "PATCH"):
            return ImportantDateCreateSerializer
        return ImportantDateSerializer

    def get_queryset(self):
        user = self.request.user
        relationship = (
            getattr(user, "relationship_as_partner1", None)
            or getattr(user, "relationship_as_partner2", None)
        )
        if not relationship:
            return ImportantDate.objects.none()
        return ImportantDate.objects.filter(relationship=relationship)


class UpcomingImportantDateListView(generics.ListAPIView):
    serializer_class = ImportantDateSerializer

    def get_queryset(self):
        from django.utils.timezone import now

        user = self.request.user
        relationship = (
            getattr(user, "relationship_as_partner1", None)
            or getattr(user, "relationship_as_partner2", None)
        )
        if not relationship:
            return ImportantDate.objects.none()

        today = now().date()
        dates = ImportantDate.objects.filter(relationship=relationship)
        return sorted(dates, key=lambda d: d.days_until)[:5]
