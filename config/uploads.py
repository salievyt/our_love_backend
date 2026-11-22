"""Правила для загружаемых файлов: случайные имена, лимиты размера, допустимые форматы."""
import os
import uuid

from django.core.exceptions import ValidationError
from django.core.validators import FileExtensionValidator
from django.utils.deconstruct import deconstructible

MAX_IMAGE_MB = 15
MAX_AUDIO_MB = 25

AUDIO_EXTENSIONS = ("m4a", "aac", "mp3", "wav", "ogg", "oga", "opus", "caf", "3gp")


@deconstructible
class UploadTo:
    """Путь вида "<prefix>/<uuid>.<ext>": имя файла от клиента не используется,
    поэтому пути нельзя угадать (клиенты всегда шлют "photo.jpg")."""

    def __init__(self, prefix):
        self.prefix = prefix.strip("/")

    def __call__(self, instance, filename):
        ext = os.path.splitext(filename)[1].lower()[:10]
        return f"{self.prefix}/{uuid.uuid4().hex}{ext}"

    def __eq__(self, other):
        return isinstance(other, UploadTo) and self.prefix == other.prefix


@deconstructible
class MaxFileSizeValidator:
    def __init__(self, max_mb):
        self.max_mb = max_mb

    def __call__(self, file):
        if file.size > self.max_mb * 1024 * 1024:
            raise ValidationError(f"Файл больше {self.max_mb} МБ")

    def __eq__(self, other):
        return isinstance(other, MaxFileSizeValidator) and self.max_mb == other.max_mb


IMAGE_VALIDATORS = [MaxFileSizeValidator(MAX_IMAGE_MB)]
AUDIO_VALIDATORS = [
    FileExtensionValidator(AUDIO_EXTENSIONS),
    MaxFileSizeValidator(MAX_AUDIO_MB),
]
