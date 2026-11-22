from datetime import timedelta

from django.utils.timezone import now

from apps.accounts.invites import ALPHABET, CODE_LENGTH
from apps.accounts.models import InviteCode, Relationship, User
from apps.accounts.testing import PairTestCase

STRONG = "velvet-orbit-42"


class InviteFlowTests(PairTestCase):
    def register(self, username, code=None, password=STRONG):
        data = {"username": username, "password": password}
        if code is not None:
            data["invite_code"] = code
        return self.client.post("/api/v1/auth/register/", data, format="json")

    def generate(self, user):
        self.login(user)
        r = self.client.post("/api/v1/auth/invite/generate/")
        self.client.force_authenticate(None)
        return r

    def test_user_without_pair_gets_strong_code(self):
        r = self.generate(self.eve)
        self.assertEqual(r.status_code, 201, r.data)
        code = r.data["code"]
        self.assertEqual(len(code), CODE_LENGTH)
        self.assertTrue(set(code) <= set(ALPHABET))

    def test_user_in_pair_cannot_generate(self):
        self.assertEqual(self.generate(self.alice).status_code, 400)

    def test_new_code_invalidates_previous_one(self):
        first = self.generate(self.eve).data["code"]
        self.generate(self.eve)
        self.assertEqual(self.register("newbie", first).status_code, 400)

    def test_register_with_code_creates_pair(self):
        code = self.generate(self.eve).data["code"]
        r = self.register("frank", code.lower())  # регистр и пробелы не важны
        self.assertEqual(r.status_code, 201, r.data)
        frank = User.objects.get(username="frank")
        rel = Relationship.objects.get(partner2=frank)
        self.assertEqual(rel.partner1, self.eve)
        self.assertTrue(InviteCode.objects.get(code=code).is_used)

    def test_invalid_code_creates_nothing(self):
        r = self.register("ghost", "WRONGCODE123")
        self.assertEqual(r.status_code, 400)
        self.assertIn("invite_code", r.data)
        self.assertFalse(User.objects.filter(username="ghost").exists())

    def test_code_cannot_be_reused(self):
        code = self.generate(self.eve).data["code"]
        self.assertEqual(self.register("frank", code).status_code, 201)
        self.assertEqual(self.register("grace", code).status_code, 400)
        self.assertFalse(User.objects.filter(username="grace").exists())

    def test_expired_code_is_rejected(self):
        code = self.generate(self.eve).data["code"]
        InviteCode.objects.filter(code=code).update(expires_at=now() - timedelta(seconds=1))
        self.assertEqual(self.register("frank", code).status_code, 400)

    def test_existing_user_accepts_code(self):
        code = self.generate(self.eve).data["code"]
        heidi = User.objects.create_user("heidi", password=STRONG)
        self.login(heidi)
        r = self.client.post("/api/v1/auth/invite/accept/", {"invite_code": code}, format="json")
        self.assertEqual(r.status_code, 201, r.data)
        self.assertEqual(r.data["partner1"]["username"], "eve")
        self.assertEqual(r.data["partner2"]["username"], "heidi")

    def test_cannot_accept_own_code_or_join_second_pair(self):
        code = self.generate(self.eve).data["code"]
        self.login(self.eve)
        self.assertEqual(self.client.post("/api/v1/auth/invite/accept/", {"invite_code": code}).status_code, 400)
        self.login(self.alice)  # уже в паре
        self.assertEqual(self.client.post("/api/v1/auth/invite/accept/", {"invite_code": code}).status_code, 400)
        self.assertEqual(Relationship.objects.filter(partner1=self.eve).count(), 0)
