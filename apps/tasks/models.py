import uuid

from django.db import models


class TaskTemplate(models.Model):
    title = models.CharField(max_length=200)
    description = models.TextField()
    emoji = models.CharField(max_length=10, default="🎯")

    class Meta:
        verbose_name = "Шаблон задания"
        verbose_name_plural = "Шаблоны заданий"

    def __str__(self):
        return self.title


class DailyTask(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    relationship = models.ForeignKey(
        "accounts.Relationship",
        on_delete=models.CASCADE,
        null=True,
        blank=True,
        related_name="tasks",
    )
    title = models.CharField(max_length=200)
    description = models.TextField(blank=True)
    emoji = models.CharField(max_length=10, default="🎯")
    is_completed = models.BooleanField(default=False)
    completed_at = models.DateTimeField(null=True, blank=True)
    photo = models.ImageField(upload_to="task_photos/", null=True, blank=True)
    date_assigned = models.DateField()
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "Ежедневное задание"
        verbose_name_plural = "Ежедневные задания"
        ordering = ("-date_assigned",)

    def __str__(self):
        return f"{self.title} ({self.date_assigned})"
