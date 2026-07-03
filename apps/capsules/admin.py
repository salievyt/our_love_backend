from django.contrib import admin

from apps.capsules.models import TimeCapsule


@admin.register(TimeCapsule)
class TimeCapsuleAdmin(admin.ModelAdmin):
    list_display = ("title", "open_date", "is_opened", "created_at")
    list_filter = ("is_opened", "open_date")
    search_fields = ("title", "message")
