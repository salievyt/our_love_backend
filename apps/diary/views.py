from django.utils.timezone import now
from rest_framework import generics
from rest_framework.parsers import FormParser, MultiPartParser

from apps.accounts.scoping import get_relationship, require_relationship
from apps.diary.models import DiaryEntry
from apps.diary.serializers import DiaryEntryCreateSerializer, DiaryEntrySerializer


def pair_entries(user):
    relationship = get_relationship(user)
    if relationship is None:
        return DiaryEntry.objects.none()
    return DiaryEntry.objects.filter(relationship=relationship)


class DiaryEntryListCreateView(generics.ListCreateAPIView):
    serializer_class = DiaryEntrySerializer
    parser_classes = (MultiPartParser, FormParser)
    filterset_fields = ("date",)
    search_fields = ("note",)

    def get_queryset(self):
        return pair_entries(self.request.user).order_by("-date")

    def get_serializer_class(self):
        if self.request.method == "POST":
            return DiaryEntryCreateSerializer
        return DiaryEntrySerializer

    def perform_create(self, serializer):
        serializer.save(relationship=require_relationship(self.request.user))


class DiaryEntryDetailView(generics.RetrieveUpdateDestroyAPIView):
    serializer_class = DiaryEntrySerializer
    parser_classes = (MultiPartParser, FormParser)

    def get_queryset(self):
        return pair_entries(self.request.user)


class TodayDiaryEntryView(generics.RetrieveAPIView):
    serializer_class = DiaryEntrySerializer

    def get_object(self):
        relationship = require_relationship(self.request.user)
        entry, _ = DiaryEntry.objects.get_or_create(relationship=relationship, date=now().date())
        return entry
