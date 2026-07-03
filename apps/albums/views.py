from django.db.models import Q
from rest_framework import generics, permissions, status
from rest_framework.parsers import FormParser, MultiPartParser
from rest_framework.response import Response

from apps.albums.models import Collage, Photo
from apps.albums.serializers import CollageSerializer, PhotoSerializer


class PhotoListCreateView(generics.ListCreateAPIView):
    serializer_class = PhotoSerializer
    parser_classes = (MultiPartParser, FormParser)
    filterset_fields = ("year_tag",)

    def get_queryset(self):
        user = self.request.user
        return Photo.objects.filter(
            Q(uploaded_by=user)
            | Q(
                uploaded_by__relationship_as_partner1__partner2=user
            )
            | Q(
                uploaded_by__relationship_as_partner2__partner1=user
            )
        ).order_by("-created_at")

    def perform_create(self, serializer):
        serializer.save(uploaded_by=self.request.user)


class PhotoDetailView(generics.RetrieveUpdateDestroyAPIView):
    serializer_class = PhotoSerializer
    parser_classes = (MultiPartParser, FormParser)

    def get_queryset(self):
        user = self.request.user
        return Photo.objects.filter(
            Q(uploaded_by=user)
            | Q(
                uploaded_by__relationship_as_partner1__partner2=user
            )
            | Q(
                uploaded_by__relationship_as_partner2__partner1=user
            )
        )


class CollageListCreateView(generics.ListCreateAPIView):
    serializer_class = CollageSerializer

    def get_queryset(self):
        user = self.request.user
        return Collage.objects.filter(
            Q(photos__uploaded_by=user)
            | Q(
                photos__uploaded_by__relationship_as_partner1__partner2=user
            )
            | Q(
                photos__uploaded_by__relationship_as_partner2__partner1=user
            )
        ).distinct().order_by("-created_at")


class CollageDetailView(generics.RetrieveDestroyAPIView):
    serializer_class = CollageSerializer

    def get_queryset(self):
        user = self.request.user
        return Collage.objects.filter(
            Q(photos__uploaded_by=user)
            | Q(
                photos__uploaded_by__relationship_as_partner1__partner2=user
            )
            | Q(
                photos__uploaded_by__relationship_as_partner2__partner1=user
            )
        ).distinct()
