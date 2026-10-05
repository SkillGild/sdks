"""Response shapes returned by the SkillGild API.

They are ``TypedDict`` types, so values stay plain dictionaries at runtime while editors
and type checkers know their keys. Fields marked ``NotRequired`` are absent on some
endpoints; the API may add new keys at any time.
"""

from __future__ import annotations

import sys
from typing import Any, Dict, List, Optional

if sys.version_info >= (3, 11):
    from typing import Literal, NotRequired, TypedDict
else:  # pragma: no cover
    from typing_extensions import Literal, NotRequired, TypedDict


class Media(TypedDict):
    kind: str
    url: str
    alt: str
    poster_url: NotRequired[str]
    preview_url: NotRequired[str]
    credit: NotRequired[str]
    width: NotRequired[int]
    height: NotRequired[int]


class FAQ(TypedDict):
    q: str
    a: str


class SkillPageContent(TypedDict, total=False):
    what_you_get: List[str]
    good_fit: List[str]
    not_for: List[str]
    requirements: List[str]
    faqs: List[FAQ]


class SkillTool(TypedDict):
    name: str
    title: str
    description: str
    input_schema: Dict[str, Any]
    example_input: NotRequired[Dict[str, Any]]


class Skill(TypedDict):
    id: str
    slug: str
    name: str
    description: str
    distribution_mode: str
    access_tier: Literal["free", "paid"]
    price_amount_minor: int
    price_currency: str
    included_in_pro: bool
    free_runs_per_month: int
    current_version: str
    input_schema: Dict[str, Any]
    runtime_type: Literal["prompt_pipeline", "hybrid_tools"]
    tools: NotRequired[List[SkillTool]]
    visibility: NotRequired[str]
    updated_at: NotRequired[str]
    category: NotRequired[str]
    category_slug: NotRequired[str]
    creator_name: NotRequired[str]
    source_url: NotRequired[str]
    license: NotRequired[str]
    attribution: NotRequired[str]
    cover: NotRequired[Media]
    media: NotRequired[List[Media]]
    page: NotRequired[SkillPageContent]


class SkillSummary(TypedDict):
    id: str
    slug: str
    name: str
    description: str
    category: str
    creator_name: str
    visibility: str
    access_tier: Literal["free", "paid"]
    price_amount_minor: int
    price_currency: str
    free_runs_per_month: int
    updated_at: str
    cover: NotRequired[Media]


class SkillPage(TypedDict):
    items: List[Skill]
    next_cursor: NotRequired[str]


class Category(TypedDict):
    id: str
    parent_id: Optional[str]
    slug: str
    name: str
    short_description: str
    sort_order: int


class Tag(TypedDict):
    id: str
    slug: str
    name: str
    color: str
    skill_count: int


class Collection(TypedDict):
    id: str
    slug: str
    name: str
    description: str
    image_url: str
    skill_count: int
    updated_at: str


class CollectionDetail(TypedDict):
    id: str
    slug: str
    name: str
    description: str
    image_url: str
    skills: List[SkillSummary]


class Usage(TypedDict):
    used: int
    limit: Optional[int]
    reset_at: str
    access_tier: str


class SkillRun(TypedDict):
    execution_id: str
    skill_id: str
    skill_slug: str
    version: str
    output: str
    usage: Usage


class AgentSession(TypedDict):
    session_id: str
    skill_id: str
    skill_slug: str
    skill_name: str
    version: str
    expires_at: str
    max_tool_calls: int
    tool_calls_used: int
    resumed: bool
    guide: str
    tools: List[SkillTool]
    usage: NotRequired[Usage]


class ToolCallResult(TypedDict):
    session_id: str
    tool: str
    result: Any
    tool_calls_used: int
    tool_calls_remaining: int
    expires_at: str


class Account(TypedDict):
    id: str
    email: str
    name: str
    auth_type: Literal["api_key", "user_jwt"]


class DeviceAuthorization(TypedDict):
    device_code: str
    user_code: str
    verification_uri: str
    expires_at: str
    interval_seconds: int


class DeviceToken(TypedDict):
    status: Literal["authorization_pending", "authorized"]
    access_token: NotRequired[str]
