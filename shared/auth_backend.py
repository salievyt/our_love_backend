"""
Shared JWT Authentication Backend.
Each microservice validates JWT tokens independently using the same SECRET_KEY.
No inter-service call is needed for auth — just shared secret key.
"""
import jwt
from django.conf import settings
from django.contrib.auth import get_user_model
from rest_framework.authentication import BaseAuthentication
from rest_framework.exceptions import AuthenticationFailed


class MicroserviceJWTAuthentication(BaseAuthentication):
    """
    Validates JWT tokens using the shared SECRET_KEY.
    Works independently in each microservice — no call to auth-service needed.
    """

    keyword = "Bearer"

    def authenticate(self, request):
        auth_header = request.headers.get("Authorization", "")
        if not auth_header.startswith("Bearer "):
            return None

        token = auth_header.split(" ", 1)[1]

        try:
            payload = jwt.decode(
                token,
                settings.SECRET_KEY,
                algorithms=["HS256"],
                options={"verify_exp": True},
            )
        except jwt.ExpiredSignatureError:
            raise AuthenticationFailed("Token expired")
        except jwt.InvalidTokenError:
            raise AuthenticationFailed("Invalid token")

        User = get_user_model()
        try:
            user = User.objects.get(id=payload.get("user_id"))
        except User.DoesNotExist:
            raise AuthenticationFailed("User not found")

        return (user, token)

    def authenticate_header(self, request):
        return "Bearer"
