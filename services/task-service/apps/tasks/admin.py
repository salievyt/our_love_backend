from django.contrib import admin

from apps.tasks.models import DailyTask, TaskTemplate


@admin.register(TaskTemplate)
class TaskTemplateAdmin(admin.ModelAdmin):
    list_display = ("title", "emoji")
    search_fields = ("title",)


@admin.register(DailyTask)
class DailyTaskAdmin(admin.ModelAdmin):
    list_display = ("title", "date_assigned", "is_completed")
    list_filter = ("is_completed", "date_assigned")
    search_fields = ("title",)
