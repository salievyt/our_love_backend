import uuid

from django.db import models


class MoodEntry(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    user = models.ForeignKey("accounts.User", on_delete=models.CASCADE, related_name="moods")
    mood = models.IntegerField(choices=[(1, "😢"), (2, "😔"), (3, "😐"), (4, "😊"), (5, "🥰")])
    note = models.TextField(blank=True)
    date = models.DateField()
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "Запись настроения"
        verbose_name_plural = "Записи настроения"
        unique_together = ("user", "date")
        ordering = ("-date",)

    def __str__(self):
        return f"{self.user.username}: {self.mood} на {self.date}"
