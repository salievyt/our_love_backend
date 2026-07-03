from rest_framework import generics
from rest_framework.parsers import FormParser, MultiPartParser

from apps.places.models import Place
from apps.places.serializers import PlaceCreateSerializer, PlaceSerializer


class PlaceListCreateView(generics.ListCreateAPIView):
    parser_classes = (MultiPartParser, FormParser)
    search_fields = ("title", "description")
    filterset_fields = ("emoji",)

    def get_serializer_class(self):
        if self.request.method == "POST":
            return PlaceCreateSerializer
        return PlaceSerializer

    def get_queryset(self):
        user = self.request.user
        return Place.objects.filter(
            models.Q(created_by=user)
            | models.Q(
                created_by__relationship_as_partner1__partner2=user
            )
            | models.Q(
                created_by__relationship_as_partner2__partner1=user
            )
        ).order_by("-created_at")

    def perform_create(self, serializer):
        serializer.save(created_by=self.request.user)


class PlaceDetailView(generics.RetrieveUpdateDestroyAPIView):
    parser_classes = (MultiPartParser, FormParser)

    def get_serializer_class(self):
        if self.request.method in ("PUT", "PATCH"):
            return PlaceCreateSerializer
        return PlaceSerializer

    def get_queryset(self):
        user = self.request.user
        return Place.objects.filter(
            models.Q(created_by=user)
            | models.Q(
                created_by__relationship_as_partner1__partner2=user
            )
            | models.Q(
                created_by__relationship_as_partner2__partner1=user
            )
        )
