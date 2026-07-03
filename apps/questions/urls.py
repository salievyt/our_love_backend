from django.urls import path

from apps.questions import views

urlpatterns = [
    path("", views.QuestionListCreateView.as_view(), name="question-list"),
    path("today/", views.TodayQuestionView.as_view(), name="question-today"),
    path("generate/", views.generate_question_view, name="question-generate"),
    path("templates/", views.QuestionTemplateListView.as_view(), name="question-templates"),
    path("<uuid:pk>/", views.QuestionDetailView.as_view(), name="question-detail"),
    path("<uuid:question_pk>/answers/", views.QuestionAnswerListView.as_view(), name="answer-list"),
    path("<uuid:question_pk>/answers/create/", views.AnswerCreateView.as_view(), name="answer-create"),
]
