import uuid

from django.db import models

from config.uploads import IMAGE_VALIDATORS, UploadTo


class Place(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    title = models.CharField(max_length=200)
    description = models.TextField(blank=True)
    emoji = models.CharField(max_length=10, default="❤️")
    latitude = models.FloatField()
    longitude = models.FloatField()
    photo = models.ImageField(
        upload_to=UploadTo("places"), validators=IMAGE_VALIDATORS, null=True, blank=True
    )
    created_by = models.ForeignKey(
        "accounts.User", on_delete=models.SET_NULL, null=True, blank=True
    )
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "Место"
        verbose_name_plural = "Места"
        ordering = ("-created_at",)

    def __str__(self):
        return f"{self.emoji} {self.title}"
