from django.utils.timezone import now
from rest_framework import generics, status
from rest_framework.response import Response

from apps.moods.models import MoodEntry
from apps.moods.serializers import MoodEntryCreateSerializer, MoodEntrySerializer


class MoodEntryListCreateView(generics.ListCreateAPIView):
    filterset_fields = ("date", "mood")

    def get_serializer_class(self):
        if self.request.method == "POST":
            return MoodEntryCreateSerializer
        return MoodEntrySerializer

    def get_queryset(self):
        return MoodEntry.objects.filter(user=self.request.user).order_by("-date")

    def perform_create(self, serializer):
        serializer.save(user=self.request.user)


class MoodEntryDetailView(generics.RetrieveUpdateDestroyAPIView):
    def get_serializer_class(self):
        if self.request.method in ("PUT", "PATCH"):
            return MoodEntryCreateSerializer
        return MoodEntrySerializer

    def get_queryset(self):
        return MoodEntry.objects.filter(user=self.request.user)


class TodayMoodView(generics.RetrieveUpdateAPIView):
    serializer_class = MoodEntrySerializer

    def get_object(self):
        today = now().date()
        mood, created = MoodEntry.objects.get_or_create(
            user=self.request.user,
            date=today,
            defaults={"mood": 3},
        )
        return mood

    def patch(self, request, *args, **kwargs):
        mood = self.get_object()
        serializer = MoodEntryCreateSerializer(
            mood, data=request.data, partial=True, context={"request": request}
        )
        serializer.is_valid(raise_exception=True)
        serializer.save()
        return Response(MoodEntrySerializer(mood).data)


class MoodStatsView(generics.RetrieveAPIView):
    def get(self, request, *args, **kwargs):
        moods = MoodEntry.objects.filter(user=request.user)
        total = moods.count()
        if total == 0:
            return Response(
                {
                    "total": 0,
                    "average": 0,
                    "good_days": 0,
                    "bad_days": 0,
                    "by_mood": {},
                }
            )

        by_mood = {}
        for i in range(1, 6):
            by_mood[str(i)] = moods.filter(mood=i).count()

        return Response(
            {
                "total": total,
                "average": round(sum(m.mood for m in moods) / total, 1),
                "good_days": moods.filter(mood__gte=4).count(),
                "bad_days": moods.filter(mood__lte=2).count(),
                "by_mood": by_mood,
            }
        )
