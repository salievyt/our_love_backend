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
    # Partner Search (Tinder-like)
    path("partner/profile/", views.PartnerProfileView.as_view(), name="partner_profile"),
    path("partner/profile/create/", views.PartnerProfileCreateView.as_view(), name="partner_profile_create"),
    path("partner/profile/delete/", views.PartnerProfileDeleteView.as_view(), name="partner_profile_delete"),
    path("partner/search/", views.PartnerSearchView.as_view(), name="partner_search"),
    path("partner/like/", views.PartnerLikeView.as_view(), name="partner_like"),
    path("partner/matches/", views.PartnerMatchesView.as_view(), name="partner_matches"),
]
