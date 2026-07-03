import uuid

from django.db import models


class ImportantDate(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    relationship = models.ForeignKey(
        "accounts.Relationship", on_delete=models.CASCADE, related_name="important_dates"
    )
    title = models.CharField(max_length=200)
    date = models.DateField()
    emoji = models.CharField(max_length=10, default="❤️")
    is_annually = models.BooleanField(default=True)
    created_by = models.ForeignKey(
        "accounts.User", on_delete=models.SET_NULL, null=True, blank=True
    )
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "Важная дата"
        verbose_name_plural = "Важные даты"
        ordering = ("date",)

    @property
    def days_until(self):
        from datetime import date
        from django.utils.timezone import now

        today = now().date()
        if self.is_annually:
            next_date = date(today.year, self.date.month, self.date.day)
            if next_date < today:
                next_date = date(today.year + 1, self.date.month, self.date.day)
            return (next_date - today).days
        return (self.date - today).days

    def __str__(self):
        return f"{self.emoji} {self.title} ({self.date})"
