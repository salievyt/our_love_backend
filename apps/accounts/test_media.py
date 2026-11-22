import io
import shutil
import tempfile
import time
from datetime import timedelta
from types import SimpleNamespace
from urllib.parse import parse_qs, urlsplit

from django.core.exceptions import ValidationError
from django.core.files.uploadedfile import SimpleUploadedFile
from django.test import override_settings
from django.utils.timezone import now
from PIL import Image

from apps.accounts.testing import PairTestCase
from apps.albums.models import Photo
from apps.capsules.models import TimeCapsule
from config.media import signed_media_url
from config.uploads import MaxFileSizeValidator


def png_file(name="photo.jpg"):
    buf = io.BytesIO()
    Image.new("RGB", (2, 2), "red").save(buf, format="PNG")
    return SimpleUploadedFile(name, buf.getvalue(), content_type="image/png")


class MediaTestCase(PairTestCase):
    def setUp(self):
        self.media_root = tempfile.mkdtemp()
        self.override = override_settings(MEDIA_ROOT=self.media_root, DEBUG=False)
        self.override.enable()

    def tearDown(self):
        self.override.disable()
        shutil.rmtree(self.media_root, ignore_errors=True)


class SignedMediaTests(MediaTestCase):
    def setUp(self):
        super().setUp()
        self.name = "photos/abc.png"
        url = signed_media_url(self.name)
        self.path, query = urlsplit(url).path, parse_qs(urlsplit(url).query)
        self.exp, self.sig = query["exp"][0], query["sig"][0]

    def get(self, path=None, **params):
        query = {"exp": self.exp, "sig": self.sig, **params}
        return self.client.get(path or self.path, {k: v for k, v in query.items() if v is not None})

    def test_valid_link_redirects_to_internal_location(self):
        r = self.get()
        self.assertEqual(r.status_code, 200)
        self.assertEqual(r["X-Accel-Redirect"], "/protected-media/photos/abc.png")
        self.assertEqual(r["X-Content-Type-Options"], "nosniff")
        self.assertNotIn("Content-Type", r)

    def test_link_works_without_login(self):
        # Картинки грузятся без заголовка Authorization — доступ даёт только подпись
        self.client.logout()
        self.assertEqual(self.get().status_code, 200)

    def test_missing_or_tampered_signature_is_404(self):
        self.assertEqual(self.client.get(self.path).status_code, 404)
        self.assertEqual(self.get(sig="x" + self.sig[1:]).status_code, 404)
        self.assertEqual(self.get(sig=None).status_code, 404)

    def test_signature_is_bound_to_path(self):
        self.assertEqual(self.get(path="/media/photos/other.png").status_code, 404)

    def test_expired_link_is_404(self):
        self.assertEqual(self.get(exp=str(int(time.time()) - 1)).status_code, 404)

    def test_non_media_file_is_forced_download(self):
        url = signed_media_url("capsules/voice/x.html")
        r = self.client.get(url)
        self.assertEqual(r["Content-Disposition"], "attachment")


class UploadRulesTests(MediaTestCase):
    def test_file_name_is_random_not_client_supplied(self):
        photo = Photo.objects.create(image=png_file("photo.jpg"), uploaded_by=self.alice)
        self.assertTrue(photo.image.name.startswith("photos/"))
        self.assertNotIn("photo.jpg", photo.image.name)
        self.assertIn("sig=", photo.image.url)

    def test_size_limit(self):
        validator = MaxFileSizeValidator(1)
        validator(SimpleNamespace(size=1024 * 1024))
        with self.assertRaises(ValidationError):
            validator(SimpleNamespace(size=1024 * 1024 + 1))

    def test_html_voice_note_rejected(self):
        self.login(self.alice)
        r = self.client.post(
            "/api/v1/capsules/",
            {
                "title": "t",
                "message": "m",
                "open_date": str(now().date() + timedelta(days=30)),
                "voice_note": SimpleUploadedFile(
                    "x.html", b"<script>alert(1)</script>", content_type="text/html"
                ),
            },
            format="multipart",
        )
        self.assertEqual(r.status_code, 400)
        self.assertIn("voice_note", r.data)

    def test_audio_voice_note_accepted(self):
        self.login(self.alice)
        r = self.client.post(
            "/api/v1/capsules/",
            {
                "title": "t",
                "message": "m",
                "open_date": str(now().date() + timedelta(days=30)),
                "voice_note": SimpleUploadedFile("note.m4a", b"\x00" * 10, content_type="audio/mp4"),
            },
            format="multipart",
        )
        self.assertEqual(r.status_code, 201, r.data)

    def test_non_image_photo_rejected(self):
        self.login(self.alice)
        r = self.client.post(
            "/api/v1/albums/",
            {"image": SimpleUploadedFile("photo.jpg", b"<html></html>", content_type="image/jpeg")},
            format="multipart",
        )
        self.assertEqual(r.status_code, 400)

    def test_opened_capsule_returns_signed_url(self):
        capsule = TimeCapsule.objects.create(
            title="t", message="m", open_date=now().date() - timedelta(days=1),
            created_by=self.alice, photo=png_file(),
        )
        self.login(self.bob)
        data = self.client.post(f"/api/v1/capsules/{capsule.id}/open/").data
        self.assertTrue(data["photo_url"].startswith("http://testserver/media/capsules/"))
        self.assertIn("sig=", data["photo_url"])
        self.assertEqual(self.client.get(data["photo_url"]).status_code, 200)
