import uuid

from django.db import models


class Photo(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    image = models.ImageField(upload_to="photos/%Y/")
    caption = models.TextField(blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    year_tag = models.IntegerField(editable=False)
    latitude = models.FloatField(null=True, blank=True)
    longitude = models.FloatField(null=True, blank=True)
    uploaded_by = models.ForeignKey(
        "accounts.User", on_delete=models.SET_NULL, null=True, blank=True
    )

    class Meta:
        verbose_name = "Фотография"
        verbose_name_plural = "Фотографии"
        ordering = ("-created_at",)

    def save(self, *args, **kwargs):
        if not self.year_tag:
            from django.utils.timezone import now

            self.year_tag = now().year
        super().save(*args, **kwargs)

    def __str__(self):
        return f"Photo {self.created_at.date()}"


class Collage(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    title = models.CharField(max_length=200)
    image = models.ImageField(upload_to="collages/", null=True, blank=True)
    photos = models.ManyToManyField(Photo, related_name="collages", blank=True)
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "Коллаж"
        verbose_name_plural = "Коллажи"
        ordering = ("-created_at",)
