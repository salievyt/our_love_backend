from django.contrib import admin

from apps.moods.models import MoodEntry


@admin.register(MoodEntry)
class MoodEntryAdmin(admin.ModelAdmin):
    list_display = ("user", "mood", "date", "note")
    list_filter = ("mood", "date")
    search_fields = ("note",)
