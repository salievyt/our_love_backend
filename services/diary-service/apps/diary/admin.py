from django.contrib import admin

from apps.diary.models import DiaryEntry


@admin.register(DiaryEntry)
class DiaryEntryAdmin(admin.ModelAdmin):
    list_display = ("date", "mood_partner1", "mood_partner2", "note")
    list_filter = ("date",)
    search_fields = ("note",)
