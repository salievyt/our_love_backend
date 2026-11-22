"""Раздача медиафайлов по подписанным ссылкам.

Публичного /media/ больше нет. API отдаёт ссылки вида
/media/<path>?exp=<unix-время>&sig=<подпись>. Подпись проверяет Django,
а сам файл отдаёт nginx через X-Accel-Redirect на internal-локацию
/protected-media/. Ссылка живёт от 1 до 2 часов и одинакова в пределах часа,
поэтому клиенты могут кэшировать картинки.
"""
import mimetypes
import time
from urllib.parse import quote, urlencode

from django.conf import settings
from django.core.files.storage import FileSystemStorage
from django.core.signing import Signer
from django.http import Http404, HttpResponse
from django.utils.crypto import constant_time_compare
from django.views.static import serve

LINK_TTL_SECONDS = 3600
_signer = Signer(salt="media-link")


def _signature(name, exp):
    return _signer.signature(f"{name}:{exp}")


def signed_media_url(name):
    exp = (int(time.time()) // LINK_TTL_SECONDS + 2) * LINK_TTL_SECONDS
    query = urlencode({"exp": exp, "sig": _signature(name, exp)})
    return f"{settings.MEDIA_URL}{quote(name)}?{query}"


class SignedFileSystemStorage(FileSystemStorage):
    def url(self, name):
        if name is None:
            return None
        return signed_media_url(name.replace("\\", "/"))


def serve_media(request, path):
    exp = request.GET.get("exp", "")
    sig = request.GET.get("sig", "")
    if (
        not exp.isdigit()
        or int(exp) < time.time()
        or ".." in path.split("/")
        or not constant_time_compare(sig, _signature(path, exp))
    ):
        raise Http404

    if settings.DEBUG:
        response = serve(request, path, document_root=settings.MEDIA_ROOT)
    else:
        response = HttpResponse()
        response["X-Accel-Redirect"] = f"/protected-media/{quote(path)}"
        # Тип определяет nginx по расширению; Django свой не навязывает
        del response["Content-Type"]

    content_type, _ = mimetypes.guess_type(path)
    if not (content_type or "").startswith(("image/", "audio/")):
        response["Content-Disposition"] = "attachment"
    response["X-Content-Type-Options"] = "nosniff"
    response["Cache-Control"] = "private, max-age=3600"
    return response


def file_url(field_file, context):
    """Полная ссылка на файл для ответа API (клиенты не дописывают домен сами)."""
    if not field_file:
        return None
    url = field_file.url
    request = (context or {}).get("request")
    return request.build_absolute_uri(url) if request is not None else url
