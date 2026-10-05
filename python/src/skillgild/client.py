"""Small synchronous client for SkillGild's hosted-skill REST API."""

from __future__ import annotations

import json
from typing import Any, Dict, List, Optional, cast
from urllib.error import HTTPError, URLError
from urllib.parse import quote, urlencode, urlsplit
from urllib.request import Request, urlopen

from .types import (
    Account,
    AgentSession,
    Category,
    Collection,
    CollectionDetail,
    DeviceAuthorization,
    DeviceToken,
    Skill,
    SkillPage,
    SkillRun,
    SkillSummary,
    Tag,
    ToolCallResult,
)

DEFAULT_BASE_URL = "https://api.skillgild.dev/v1"


class SkillGildError(RuntimeError):
    """An API error. ``retry_after`` is the Retry-After delay in seconds, when the API sent one."""

    def __init__(
        self,
        message: str,
        status: int,
        code: Optional[str] = None,
        retry_after: Optional[int] = None,
    ):
        super().__init__(message)
        self.status = status
        self.code = code
        self.retry_after = retry_after


class SkillGildClient:
    """API client. Pass an API key for authenticated skill execution."""

    def __init__(
        self,
        api_key: Optional[str] = None,
        base_url: str = DEFAULT_BASE_URL,
        timeout: float = 100.0,
    ):
        parsed = urlsplit(base_url)
        if parsed.scheme not in ("http", "https") or not parsed.netloc or parsed.username or parsed.password:
            raise ValueError("base_url must be an absolute HTTP(S) URL without user information")
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.timeout = timeout

    # Public catalog

    def list_skills(
        self,
        query: Optional[str] = None,
        *,
        limit: int = 100,
        cursor: Optional[str] = None,
    ) -> SkillPage:
        """One page of the public catalog: ``{"items": [...], "next_cursor": "..."}``.

        ``next_cursor`` is absent on the last page. ``limit`` is 1 to 100.
        """
        params: Dict[str, Any] = {"limit": limit}
        if query and query.strip():
            params["q"] = query.strip()
        if cursor:
            params["cursor"] = cursor
        return cast(SkillPage, self._request("GET", "/skills?" + urlencode(params)))

    def search_skills(self, query: Optional[str] = None) -> List[Skill]:
        """The first 100 matching skills. Use ``list_skills`` to page through more."""
        return self.list_skills(query, limit=100)["items"]

    def get_skill(self, skill_id_or_slug: str) -> Skill:
        return cast(Skill, self._request("GET", "/skills/" + _segment(skill_id_or_slug)))

    def list_categories(self) -> List[Category]:
        return cast(List[Category], self._request("GET", "/categories"))

    def list_tags(self) -> List[Tag]:
        return cast(List[Tag], self._request("GET", "/tags"))

    def list_collections(self) -> List[Collection]:
        return cast(List[Collection], self._request("GET", "/collections"))

    def get_collection(self, slug: str) -> CollectionDetail:
        return cast(CollectionDetail, self._request("GET", "/collections/" + _segment(slug)))

    def featured_skills(self, slot: str = "home") -> List[SkillSummary]:
        """Skills featured in a placement slot (``"home"`` by default)."""
        return cast(List[SkillSummary], self._request("GET", "/featured-skills?" + urlencode({"slot": slot})))

    # Account and login

    def me(self) -> Account:
        """The account the API key belongs to: ``id``, ``email``, ``name``, ``auth_type``."""
        self._require_key("read the account")
        return cast(Account, self._request("GET", "/me", authenticated=True))

    def revoke_current_key(self) -> None:
        """Revoke the API key this client authenticates with. Later calls with it fail with 401."""
        self._require_key("revoke the current key")
        self._request("DELETE", "/me/api-keys/current", authenticated=True)

    def start_device_authorization(self, device_name: str, *, client_type: str = "skillgild-sdk") -> DeviceAuthorization:
        """Start a device login. Show the user ``user_code`` and ``verification_uri``, then call
        ``poll_device_authorization`` every ``interval_seconds`` until it returns an access token."""
        return cast(
            DeviceAuthorization,
            self._request("POST", "/device-authorizations", {"device_name": device_name, "client_type": client_type}),
        )

    def poll_device_authorization(self, device_code: str) -> DeviceToken:
        return cast(DeviceToken, self._request("POST", "/device-authorizations/token", {"device_code": device_code}))

    # Running skills

    def run_skill(
        self,
        skill_id_or_slug: str,
        input: Dict[str, Any],
        *,
        idempotency_key: Optional[str] = None,
    ) -> SkillRun:
        self._require_key("run hosted skills")
        return cast(
            SkillRun,
            self._request(
                "POST",
                "/skills/" + _segment(skill_id_or_slug) + "/run",
                {"input": input},
                authenticated=True,
                idempotency_key=idempotency_key,
            ),
        )

    def start_session(self, skill_id_or_slug: str) -> AgentSession:
        """Start (or resume) a session for a hybrid_tools skill. A new session uses one run."""
        self._require_key("start a skill session")
        return cast(
            AgentSession,
            self._request("POST", "/skills/" + _segment(skill_id_or_slug) + "/sessions", {}, authenticated=True),
        )

    def call_tool(self, session_id: str, tool: str, input: Dict[str, Any]) -> ToolCallResult:
        """Call a server tool in an open session. Each call uses one of the session's calls."""
        self._require_key("call skill tools")
        return cast(
            ToolCallResult,
            self._request(
                "POST",
                "/skill-sessions/" + _segment(session_id) + "/tools/" + _segment(tool),
                {"input": input},
                authenticated=True,
            ),
        )

    def end_session(self, session_id: str) -> None:
        self._require_key("end a skill session")
        self._request("DELETE", "/skill-sessions/" + _segment(session_id), authenticated=True)

    def _require_key(self, action: str) -> None:
        if not self.api_key:
            raise ValueError("api_key is required to " + action)

    def _request(
        self,
        method: str,
        path: str,
        body: Optional[Dict[str, Any]] = None,
        authenticated: bool = False,
        idempotency_key: Optional[str] = None,
    ) -> Any:
        headers = {"Accept": "application/json", "User-Agent": "skillgild-python"}
        payload = None
        if body is not None:
            payload = json.dumps(body).encode("utf-8")
            headers["Content-Type"] = "application/json"
        if authenticated:
            headers["Authorization"] = "Bearer " + str(self.api_key)
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        request = Request(self.base_url + path, data=payload, headers=headers, method=method)
        try:
            with urlopen(request, timeout=self.timeout) as response:
                raw = response.read()
                if response.status == 204 or not raw:
                    return None
                try:
                    envelope = json.loads(raw.decode("utf-8"))
                except (ValueError, UnicodeDecodeError):
                    raise SkillGildError(
                        "SkillGild API returned an unexpected response", response.status
                    ) from None
        except HTTPError as error:
            try:
                envelope = json.loads(error.read().decode("utf-8"))
            except (ValueError, UnicodeDecodeError):
                envelope = {}
            detail = envelope.get("error") if isinstance(envelope, dict) else None
            if not isinstance(detail, dict):
                detail = {}
            raise SkillGildError(
                detail.get("message") or "SkillGild API request failed (%d)" % error.code,
                error.code,
                detail.get("code"),
                _retry_after(error.headers.get("Retry-After") if error.headers else None),
            ) from None
        except (URLError, TimeoutError) as error:
            raise SkillGildError("SkillGild API is unavailable", 0) from error
        return envelope.get("data") if isinstance(envelope, dict) else None


def _segment(value: str) -> str:
    return quote(value, safe="")


def _retry_after(value: Optional[str]) -> Optional[int]:
    try:
        return int(value) if value is not None else None
    except ValueError:
        return None
