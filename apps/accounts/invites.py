"""Invite-коды: создание пары по коду партнёра."""
import secrets
from datetime import timedelta

from django.db import transaction
from django.utils.timezone import now
from rest_framework.exceptions import ValidationError

from apps.accounts.models import InviteCode, Relationship
from apps.accounts.scoping import get_relationship

# Без похожих символов (0/O, 1/I/L), чтобы код было легко продиктовать и ввести
ALPHABET = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
CODE_LENGTH = 12  # ~59 бит: перебором не подобрать даже без лимитов
CODE_TTL = timedelta(days=7)
INVALID_CODE = "Код приглашения недействителен или истёк"


def generate_code():
    return "".join(secrets.choice(ALPHABET) for _ in range(CODE_LENGTH))


def normalize(code):
    return "".join((code or "").split()).replace("-", "").upper()


def issue_invite(user):
    """Новый код для пользователя без пары; прежние неиспользованные коды гасятся."""
    if get_relationship(user) is not None:
        raise ValidationError({"detail": "Вы уже в паре"})
    with transaction.atomic():
        InviteCode.objects.filter(created_by=user, is_used=False).update(is_used=True)
        return InviteCode.objects.create(
            code=generate_code(), created_by=user, expires_at=now() + CODE_TTL
        )


def accept_invite(user, code):
    """Создаёт пару создателя кода и user. Вызывать внутри transaction.atomic()."""
    invite = (
        InviteCode.objects.select_for_update()
        .select_related("created_by")
        .filter(code=normalize(code), is_used=False, expires_at__gt=now())
        .first()
    )
    if invite is None or invite.created_by_id == user.pk:
        raise ValidationError({"invite_code": [INVALID_CODE]})
    if get_relationship(invite.created_by) is not None or get_relationship(user) is not None:
        raise ValidationError({"invite_code": ["Один из партнёров уже состоит в паре"]})

    relationship = Relationship.objects.create(
        partner1=invite.created_by,
        partner2=user,
        relationship_start_date=now().date(),
    )
    invite.is_used = True
    invite.relationship = relationship
    invite.save(update_fields=["is_used", "relationship"])
    return relationship
