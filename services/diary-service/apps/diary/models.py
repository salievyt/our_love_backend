import uuid

from django.db import models


class DiaryEntry(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    date = models.DateField(unique=True)
    note = models.TextField(blank=True)
    photo = models.ImageField(upload_to="diary/", null=True, blank=True)
    mood_partner1 = models.IntegerField(null=True, blank=True)
    mood_partner2 = models.IntegerField(null=True, blank=True)
    question_answer_partner1 = models.ForeignKey(
        "questions.Answer",
        on_delete=models.SET_NULL,
        null=True,
        blank=True,
        related_name="diary_entries_as_partner1",
    )
    question_answer_partner2 = models.ForeignKey(
        "questions.Answer",
        on_delete=models.SET_NULL,
        null=True,
        blank=True,
        related_name="diary_entries_as_partner2",
    )
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "Запись дневника"
        verbose_name_plural = "Записи дневника"
        ordering = ("-date",)

    def __str__(self):
        return f"Дневник от {self.date}"
