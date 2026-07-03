import uuid

from django.db import models


class TimeCapsule(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    title = models.CharField(max_length=200)
    message = models.TextField()
    photo = models.ImageField(upload_to="capsules/", null=True, blank=True)
    voice_note = models.FileField(upload_to="capsules/voice/", null=True, blank=True)
    open_date = models.DateField()
    is_opened = models.BooleanField(default=False)
    opened_at = models.DateTimeField(null=True, blank=True)
    created_by = models.ForeignKey(
        "accounts.User", on_delete=models.SET_NULL, null=True, blank=True
    )
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "Капсула времени"
        verbose_name_plural = "Капсулы времени"
        ordering = ("open_date",)

    @property
    def is_ready_to_open(self):
        from django.utils.timezone import now

        return now().date() >= self.open_date and not self.is_opened

    def __str__(self):
        return f"⏳ {self.title} (открыть {self.open_date})"
