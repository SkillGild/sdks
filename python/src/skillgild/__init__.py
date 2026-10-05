"""Official Python client for SkillGild, the marketplace for hosted AI agent skills."""

from .client import DEFAULT_BASE_URL, SkillGildClient, SkillGildError
from . import types

__version__ = "0.1.0"
__all__ = ["DEFAULT_BASE_URL", "SkillGildClient", "SkillGildError", "types", "__version__"]
