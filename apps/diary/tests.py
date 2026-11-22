from django.utils.timezone import now

from apps.accounts.testing import PairTestCase
from apps.diary.models import DiaryEntry


class DiaryVisibilityTests(PairTestCase):
    def setUp(self):
        self.entry = DiaryEntry.objects.create(relationship=self.pair, date=now().date(), note="наш день")

    def test_stranger_sees_nothing(self):
        self.login(self.carol)
        self.assertEqual(self.results(self.client.get("/api/v1/diary/")), [])
        self.assertEqual(self.results(self.client.get("/api/v1/diary/?search=наш")), [])
        url = f"/api/v1/diary/{self.entry.id}/"
        self.assertEqual(self.client.get(url).status_code, 404)
        self.assertEqual(self.client.patch(url, {"note": "x"}).status_code, 404)
        self.assertEqual(self.client.delete(url).status_code, 404)

    def test_partner_sees_entry(self):
        self.login(self.bob)
        notes = [e["note"] for e in self.results(self.client.get("/api/v1/diary/"))]
        self.assertEqual(notes, ["наш день"])

    def test_other_pair_can_write_same_date(self):
        self.login(self.carol)
        r = self.client.post("/api/v1/diary/", {"date": str(now().date()), "note": "их день"})
        self.assertEqual(r.status_code, 201, r.data)
        self.assertEqual(DiaryEntry.objects.filter(date=now().date()).count(), 2)

    def test_same_pair_cannot_duplicate_date(self):
        self.login(self.bob)
        r = self.client.post("/api/v1/diary/", {"date": str(now().date()), "note": "ещё"})
        self.assertEqual(r.status_code, 400)

    def test_today_is_per_pair(self):
        self.login(self.carol)
        r = self.client.get("/api/v1/diary/today/")
        self.assertEqual(r.status_code, 200)
        self.assertNotEqual(r.data["id"], str(self.entry.id))
        self.login(self.alice)
        self.assertEqual(self.client.get("/api/v1/diary/today/").data["id"], str(self.entry.id))

    def test_user_without_pair(self):
        self.login(self.eve)
        self.assertEqual(self.results(self.client.get("/api/v1/diary/")), [])
        self.assertEqual(self.client.get("/api/v1/diary/today/").status_code, 400)
        self.assertEqual(self.client.post("/api/v1/diary/", {"date": "2026-01-01"}).status_code, 400)
