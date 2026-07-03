import uuid

from django.contrib.auth.models import AbstractUser
from django.db import models


class User(AbstractUser):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    avatar = models.ImageField(upload_to="avatars/", blank=True, null=True)
    phone = models.CharField(max_length=20, blank=True)

    class Meta:
        verbose_name = "Пользователь"
        verbose_name_plural = "Пользователи"

    def __str__(self):
        return self.get_full_name() or self.username


class Relationship(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    partner1 = models.OneToOneField(
        User, on_delete=models.CASCADE, related_name="relationship_as_partner1"
    )
    partner2 = models.OneToOneField(
        User, on_delete=models.CASCADE, related_name="relationship_as_partner2"
    )
    partner1_name = models.CharField("Имя партнёра 1", max_length=100, default="Ты")
    partner2_name = models.CharField("Имя партнёра 2", max_length=100, default="Второй половинка")
    partner1_emoji = models.CharField(max_length=10, default="💙")
    partner2_emoji = models.CharField(max_length=10, default="💖")
    relationship_start_date = models.DateField("Дата начала отношений")
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "Пара"
        verbose_name_plural = "Пары"

    def __str__(self):
        return f"{self.partner1_name} & {self.partner2_name}"

    @property
    def days_together(self):
        from django.utils.timezone import now

        delta = now().date() - self.relationship_start_date
        return max(delta.days, 0)


class InviteCode(models.Model):
    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    relationship = models.ForeignKey(
        Relationship, on_delete=models.CASCADE, related_name="invite_codes"
    )
    code = models.CharField(max_length=8, unique=True)
    created_by = models.ForeignKey(User, on_delete=models.CASCADE)
    is_used = models.BooleanField(default=False)
    created_at = models.DateTimeField(auto_now_add=True)
    expires_at = models.DateTimeField()

    class Meta:
        verbose_name = "Код приглашения"
        verbose_name_plural = "Коды приглашения"
