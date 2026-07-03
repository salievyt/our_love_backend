from django.contrib import admin

from apps.questions.models import Answer, QuestionOfTheDay, QuestionTemplate


@admin.register(QuestionTemplate)
class QuestionTemplateAdmin(admin.ModelAdmin):
    list_display = ("text",)
    search_fields = ("text",)


@admin.register(QuestionOfTheDay)
class QuestionOfTheDayAdmin(admin.ModelAdmin):
    list_display = ("question", "date_assigned")
    list_filter = ("date_assigned",)
    search_fields = ("question",)


@admin.register(Answer)
class AnswerAdmin(admin.ModelAdmin):
    list_display = ("question", "user", "created_at")
    list_filter = ("created_at",)
