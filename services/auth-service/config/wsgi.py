import os
import sys
from pathlib import Path

# Добавляем shared в PYTHONPATH
sys.path.insert(0, str(Path(__file__).resolve().parent.parent.parent.parent / "shared"))

from django.core.wsgi import get_wsgi_application

os.environ.setdefault("DJANGO_SETTINGS_MODULE", "config.settings")

application = get_wsgi_application()
