from datetime import timedelta

from django.utils.timezone import now

from apps.accounts.testing import PairTestCase
from apps.capsules.models import TimeCapsule


class CapsuleVisibilityTests(PairTestCase):
    def setUp(self):
        today = now().date()
        self.sealed = TimeCapsule.objects.create(
            title="Секрет", message="secret text", open_date=today + timedelta(days=30),
            created_by=self.alice,
        )
        self.ready = TimeCapsule.objects.create(
            title="Готова", message="ready text", open_date=today - timedelta(days=1),
            created_by=self.alice,
        )

    def test_stranger_sees_nothing(self):
        self.login(self.carol)
        for url in ("/api/v1/capsules/", "/api/v1/capsules/active/", "/api/v1/capsules/opened/"):
            self.assertEqual(self.results(self.client.get(url)), [], url)
        self.assertEqual(self.client.get(f"/api/v1/capsules/{self.sealed.id}/").status_code, 404)

    def test_stranger_cannot_modify_open_or_delete(self):
        self.login(self.carol)
        url = f"/api/v1/capsules/{self.ready.id}/"
        self.assertEqual(self.client.patch(url, {"title": "x"}).status_code, 404)
        self.assertEqual(self.client.delete(url).status_code, 404)
        self.assertEqual(self.client.post(url + "open/").status_code, 404)
        self.ready.refresh_from_db()
        self.assertFalse(self.ready.is_opened)

    def test_partner_sees_capsule_but_not_sealed_content(self):
        self.login(self.bob)
        ids = {c["id"] for c in self.results(self.client.get("/api/v1/capsules/"))}
        self.assertEqual(ids, {str(self.sealed.id), str(self.ready.id)})
        data = self.client.get(f"/api/v1/capsules/{self.sealed.id}/").data
        self.assertEqual(data["title"], "Секрет")
        self.assertIsNone(data["message"])

    def test_author_cannot_read_sealed_content_either(self):
        self.login(self.alice)
        self.assertIsNone(self.client.get(f"/api/v1/capsules/{self.sealed.id}/").data["message"])

    def test_partner_cannot_edit_or_delete_authors_capsule(self):
        self.login(self.bob)
        url = f"/api/v1/capsules/{self.sealed.id}/"
        self.assertEqual(self.client.patch(url, {"title": "x"}).status_code, 404)
        self.assertEqual(self.client.delete(url).status_code, 404)

    def test_open_date_cannot_be_moved_to_past(self):
        self.login(self.alice)
        r = self.client.patch(
            f"/api/v1/capsules/{self.sealed.id}/", {"open_date": "2000-01-01"}, format="multipart"
        )
        self.assertEqual(r.status_code, 400)

    def test_sealed_capsule_cannot_be_opened(self):
        self.login(self.alice)
        self.assertEqual(self.client.post(f"/api/v1/capsules/{self.sealed.id}/open/").status_code, 400)

    def test_partner_opens_ready_capsule_and_reads_it(self):
        self.login(self.bob)
        r = self.client.post(f"/api/v1/capsules/{self.ready.id}/open/")
        self.assertEqual(r.status_code, 200)
        self.assertEqual(r.data["message"], "ready text")

    def test_open_unknown_capsule_is_404(self):
        self.login(self.alice)
        r = self.client.post("/api/v1/capsules/00000000-0000-0000-0000-000000000000/open/")
        self.assertEqual(r.status_code, 404)
