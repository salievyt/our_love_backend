from rest_framework import generics
from rest_framework.parsers import FormParser, MultiPartParser

from apps.diary.models import DiaryEntry
from apps.diary.serializers import DiaryEntryCreateSerializer, DiaryEntrySerializer


class DiaryEntryListCreateView(generics.ListCreateAPIView):
    serializer_class = DiaryEntrySerializer
    parser_classes = (MultiPartParser, FormParser)
    filterset_fields = ("date",)
    search_fields = ("note",)

    def get_queryset(self):
        return DiaryEntry.objects.all().order_by("-date")

    def get_serializer_class(self):
        if self.request.method == "POST":
            return DiaryEntryCreateSerializer
        return DiaryEntrySerializer

    def perform_create(self, serializer):
        serializer.save()


class DiaryEntryDetailView(generics.RetrieveUpdateDestroyAPIView):
    serializer_class = DiaryEntrySerializer
    parser_classes = (MultiPartParser, FormParser)

    def get_queryset(self):
        return DiaryEntry.objects.all()


class TodayDiaryEntryView(generics.RetrieveAPIView):
    serializer_class = DiaryEntrySerializer

    def get_object(self):
        from django.utils.timezone import now

        today = now().date()
        entry, created = DiaryEntry.objects.get_or_create(date=today)
        return entry
