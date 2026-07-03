from django.contrib import admin

from apps.places.models import Place


@admin.register(Place)
class PlaceAdmin(admin.ModelAdmin):
    list_display = ("title", "emoji", "latitude", "longitude", "created_at")
    search_fields = ("title", "description")
