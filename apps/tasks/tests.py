from django.utils.timezone import now

from apps.accounts.testing import PairTestCase
from apps.tasks.models import DailyTask, TaskTemplate


class TaskVisibilityTests(PairTestCase):
    def setUp(self):
        self.task = DailyTask.objects.create(
            relationship=self.pair, title="Закат", date_assigned=now().date()
        )

    def test_stranger_cannot_touch_task(self):
        self.login(self.carol)
        url = f"/api/v1/tasks/{self.task.id}/"
        self.assertEqual(self.results(self.client.get("/api/v1/tasks/")), [])
        self.assertEqual(self.client.get(url).status_code, 404)
        self.assertEqual(self.client.patch(url, {"title": "фишинг"}).status_code, 404)
        self.assertEqual(self.client.delete(url).status_code, 404)
        self.assertEqual(self.client.patch(url + "complete/").status_code, 404)
        self.task.refresh_from_db()
        self.assertEqual(self.task.title, "Закат")
        self.assertFalse(self.task.is_completed)

    def test_partner_completes_task_via_post_and_put(self):
        self.login(self.bob)
        r = self.client.post(f"/api/v1/tasks/{self.task.id}/complete/")
        self.assertEqual(r.status_code, 200, r.data)
        self.assertTrue(r.data["is_completed"])
        self.assertEqual(self.client.put(f"/api/v1/tasks/{self.task.id}/complete/").status_code, 200)

    def test_today_task_is_per_pair(self):
        TaskTemplate.objects.create(title="Шаблон", description="d", emoji="🎯")
        self.login(self.carol)
        data = self.client.get("/api/v1/tasks/today/").data
        self.assertNotEqual(data["id"], str(self.task.id))
        self.login(self.alice)
        self.assertEqual(self.client.get("/api/v1/tasks/today/").data["id"], str(self.task.id))

    def test_generate_accepts_get_and_post_and_is_per_pair(self):
        TaskTemplate.objects.create(title="Шаблон", description="d", emoji="🎯")
        self.login(self.carol)
        self.assertEqual(self.client.post("/api/v1/tasks/generate/").status_code, 201)
        self.assertEqual(self.client.get("/api/v1/tasks/generate/").status_code, 400)
        self.assertEqual(DailyTask.objects.filter(relationship=self.other_pair).count(), 1)

    def test_created_task_belongs_to_own_pair(self):
        self.login(self.carol)
        r = self.client.post("/api/v1/tasks/", {"title": "Наше", "description": "", "emoji": "💕"})
        self.assertEqual(r.status_code, 201, r.data)
        self.assertEqual(DailyTask.objects.get(pk=r.data["id"]).relationship, self.other_pair)

    def test_user_without_pair(self):
        self.login(self.eve)
        self.assertEqual(self.client.get("/api/v1/tasks/today/").status_code, 400)
        self.assertEqual(self.client.post("/api/v1/tasks/", {"title": "x"}).status_code, 400)
