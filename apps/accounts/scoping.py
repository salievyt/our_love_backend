"""Ограничение видимости данных парой.

Правило проекта: общие данные пары видят только два партнёра,
личные данные — только их владелец.
"""
from django.db.models import Q
from rest_framework.exceptions import ValidationError


def get_relationship(user):
    """Пара пользователя или None."""
    return getattr(user, "relationship_as_partner1", None) or getattr(
        user, "relationship_as_partner2", None
    )


def require_relationship(user):
    """Пара пользователя; 400, если пары нет."""
    relationship = get_relationship(user)
    if relationship is None:
        raise ValidationError({"detail": "Сначала создайте пару с партнёром"})
    return relationship


def pair_q(user, field):
    """Q-фильтр: объект создан пользователем или его партнёром.

    field — имя FK на пользователя, например "created_by" или "uploaded_by".
    """
    return (
        Q(**{field: user})
        | Q(**{f"{field}__relationship_as_partner1__partner2": user})
        | Q(**{f"{field}__relationship_as_partner2__partner1": user})
    )
