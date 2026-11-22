import uuid

from django.db import models


class QuestionTemplate(models.Model):
    text = models.TextField(unique=True)

    class Meta:
        verbose_name = "Шаблон вопроса"
        verbose_name_plural = "Шаблоны вопросов"

    def __str__(self):
        return self.text[:50]


class QuestionOfTheDay(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    relationship = models.ForeignKey(
        "accounts.Relationship",
        on_delete=models.CASCADE,
        null=True,
        blank=True,
        related_name="questions",
    )
    question = models.TextField()
    date_assigned = models.DateField()
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "Вопрос дня"
        verbose_name_plural = "Вопросы дня"
        ordering = ("-date_assigned",)
        constraints = [
            models.UniqueConstraint(
                fields=("relationship", "date_assigned"),
                name="question_unique_relationship_date",
            )
        ]

    def __str__(self):
        return f"Вопрос от {self.date_assigned}: {self.question[:30]}"


class Answer(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    question = models.ForeignKey(
        QuestionOfTheDay, on_delete=models.CASCADE, related_name="answers"
    )
    user = models.ForeignKey("accounts.User", on_delete=models.CASCADE)
    text = models.TextField()
    created_at = models.DateTimeField(auto_now_add=True)

    class Meta:
        verbose_name = "Ответ"
        verbose_name_plural = "Ответы"
        unique_together = ("question", "user")

    def __str__(self):
        return f"Ответ от {self.user.username}: {self.text[:30]}"
