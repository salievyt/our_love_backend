from django.contrib import admin

from apps.albums.models import Collage, Photo


@admin.register(Photo)
class PhotoAdmin(admin.ModelAdmin):
    list_display = ("caption", "year_tag", "created_at", "uploaded_by")
    list_filter = ("year_tag",)
    search_fields = ("caption",)


@admin.register(Collage)
class CollageAdmin(admin.ModelAdmin):
    list_display = ("title", "created_at")
    search_fields = ("title",)
