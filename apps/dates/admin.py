from django.contrib import admin

from apps.dates.models import ImportantDate


@admin.register(ImportantDate)
class ImportantDateAdmin(admin.ModelAdmin):
    list_display = ("title", "date", "emoji", "is_annually", "days_until")
    list_filter = ("is_annually",)
    search_fields = ("title",)
