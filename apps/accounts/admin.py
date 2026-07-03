from django.contrib import admin

from apps.accounts.models import InviteCode, Relationship, User


@admin.register(User)
class UserAdmin(admin.ModelAdmin):
    list_display = ("username", "email", "first_name", "last_name", "is_active")
    search_fields = ("username", "email", "first_name", "last_name")
    list_filter = ("is_active", "is_staff")


@admin.register(Relationship)
class RelationshipAdmin(admin.ModelAdmin):
    list_display = ("partner1_name", "partner2_name", "relationship_start_date", "days_together")
    search_fields = ("partner1_name", "partner2_name")


@admin.register(InviteCode)
class InviteCodeAdmin(admin.ModelAdmin):
    list_display = ("code", "relationship", "is_used", "created_at", "expires_at")
    list_filter = ("is_used",)
