"""Общие фикстуры для тестов видимости данных пары."""
from datetime import date
from unittest import mock

from django.contrib.auth import get_user_model
from django.core.cache.backends.locmem import LocMemCache
from rest_framework.test import APITestCase
from rest_framework.throttling import SimpleRateThrottle

from apps.accounts.models import Relationship

User = get_user_model()


class IsolatedThrottleMixin:
    """Лимиты запросов считаются в отдельном кэше на каждый тест, а не в общем Redis."""

    def setUp(self):
        super().setUp()
        patcher = mock.patch.object(SimpleRateThrottle, "cache", LocMemCache(f"throttle-{id(self)}", {}))
        patcher.start()
        self.addCleanup(patcher.stop)


class PairTestCase(IsolatedThrottleMixin, APITestCase):
    """alice + bob — пара; carol + dave — другая пара; eve — без пары."""

    @classmethod
    def setUpTestData(cls):
        cls.alice = User.objects.create_user("alice", password="pass-alice-1")
        cls.bob = User.objects.create_user("bob", password="pass-bob-1")
        cls.carol = User.objects.create_user("carol", password="pass-carol-1")
        cls.dave = User.objects.create_user("dave", password="pass-dave-1")
        cls.eve = User.objects.create_user("eve", password="pass-eve-1")
        cls.pair = Relationship.objects.create(
            partner1=cls.alice, partner2=cls.bob, relationship_start_date=date(2024, 1, 1)
        )
        cls.other_pair = Relationship.objects.create(
            partner1=cls.carol, partner2=cls.dave, relationship_start_date=date(2024, 1, 1)
        )

    def login(self, user):
        self.client.force_authenticate(user)

    def results(self, response):
        data = response.data
        return data["results"] if isinstance(data, dict) and "results" in data else data
