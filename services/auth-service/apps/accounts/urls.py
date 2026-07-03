from django.urls import path
from rest_framework_simplejwt.views import TokenObtainPairView, TokenRefreshView, TokenBlacklistView

from apps.accounts import views

urlpatterns = [
    # JWT Auth
    path("login/", TokenObtainPairView.as_view(), name="token_obtain_pair"),
    path("refresh/", TokenRefreshView.as_view(), name="token_refresh"),
    path("logout/", TokenBlacklistView.as_view(), name="token_blacklist"),
    # Registration & Profile
    path("register/", views.RegisterView.as_view(), name="register"),
    path("me/", views.me_view, name="me"),
    path("profile/", views.UserProfileView.as_view(), name="profile"),
    # Relationship
    path("relationship/", views.RelationshipDetailView.as_view(), name="relationship"),
    path("invite/generate/", views.GenerateInviteCodeView.as_view(), name="generate_invite"),
]
