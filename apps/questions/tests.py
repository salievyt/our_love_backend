from django.utils.timezone import now

from apps.accounts.testing import PairTestCase
from apps.questions.models import Answer, QuestionOfTheDay


class QuestionVisibilityTests(PairTestCase):
    def setUp(self):
        self.question = QuestionOfTheDay.objects.create(
            relationship=self.pair, question="Q?", date_assigned=now().date()
        )
        self.alice_answer = Answer.objects.create(question=self.question, user=self.alice, text="ответ Алисы")

    def answers_url(self):
        return f"/api/v1/questions/{self.question.id}/answers/"

    def test_stranger_cannot_see_or_answer(self):
        self.login(self.carol)
        self.assertEqual(self.results(self.client.get("/api/v1/questions/")), [])
        self.assertEqual(self.client.get(f"/api/v1/questions/{self.question.id}/").status_code, 404)
        self.assertEqual(self.client.get(self.answers_url()).status_code, 404)
        r = self.client.post(self.answers_url() + "create/", {"text": "x"})
        self.assertEqual(r.status_code, 404)

    def test_partner_answer_hidden_until_own_answer(self):
        self.login(self.bob)
        self.assertEqual(self.results(self.client.get(self.answers_url())), [])
        data = self.client.get("/api/v1/questions/today/").data
        self.assertEqual(data["id"], str(self.question.id))
        self.assertEqual(data["answers"], [])
        self.assertTrue(data["partner1_answered"])
        self.assertFalse(data["partner2_answered"])

        r = self.client.post(self.answers_url() + "create/", {"text": "ответ Боба"})
        self.assertEqual(r.status_code, 201, r.data)
        texts = {a["text"] for a in self.results(self.client.get(self.answers_url()))}
        self.assertEqual(texts, {"ответ Алисы", "ответ Боба"})
        self.assertTrue(self.client.get("/api/v1/questions/today/").data["both_answered"])

    def test_duplicate_answer_is_400(self):
        self.login(self.alice)
        r = self.client.post(self.answers_url() + "create/", {"text": "снова"})
        self.assertEqual(r.status_code, 400)

    def test_today_question_is_per_pair(self):
        self.login(self.carol)
        data = self.client.get("/api/v1/questions/today/").data
        self.assertNotEqual(data["id"], str(self.question.id))
        self.assertEqual(data["answers"], [])

    def test_created_question_belongs_to_own_pair(self):
        self.login(self.carol)
        r = self.client.post("/api/v1/questions/", {"question": "Наш?", "date_assigned": "2030-01-01"})
        self.assertEqual(r.status_code, 201, r.data)
        self.assertEqual(QuestionOfTheDay.objects.get(pk=r.data["id"]).relationship, self.other_pair)
        self.login(self.alice)
        self.assertEqual(self.client.get(f"/api/v1/questions/{r.data['id']}/").status_code, 404)

    def test_duplicate_date_in_pair_is_400(self):
        self.login(self.alice)
        r = self.client.post("/api/v1/questions/", {"question": "x", "date_assigned": str(now().date())})
        self.assertEqual(r.status_code, 400)

    def test_user_without_pair(self):
        self.login(self.eve)
        self.assertEqual(self.client.get("/api/v1/questions/today/").status_code, 400)
        self.assertEqual(self.results(self.client.get("/api/v1/questions/")), [])
